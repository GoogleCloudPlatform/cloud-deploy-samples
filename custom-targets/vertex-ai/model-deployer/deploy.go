// Copyright 2023 Google LLC

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     https://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"fmt"
	aiplatform "google.golang.org/api/aiplatform/v1beta1"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/cloud-deploy-samples/custom-targets/util/clouddeploy"
	"sigs.k8s.io/yaml"

	"cloud.google.com/go/storage"
	"k8s.io/apimachinery/pkg/util/wait"
)

const aiDeployerSampleName = "clouddeploy-vertex-ai-sample"

// deployer implements the handler interface to deploy a model using the vertex AI API.`.
type deployer struct {
	gcsClient *storage.Client
	params    *params
	req       *clouddeploy.DeployRequest
}

// process processes the Deploy request.
func (d *deployer) process(ctx context.Context) error {
	fmt.Println("Processing deploy request")

	res, err := d.deploy(ctx)
	if err != nil {
		fmt.Printf("Deploy failed: %v\n", err)
		dr := &clouddeploy.DeployResult{
			ResultStatus:   clouddeploy.DeployFailed,
			FailureMessage: err.Error(),
		}
		d.addCommonMetadata(res)
		fmt.Println("Uploading failed deploy results")
		rURI, err := clouddeploy.UploadDeployResult(ctx, d.gcsClient, d.req.OutputGCSPath, dr)
		if err != nil {
			return fmt.Errorf("error uploading failed deploy results: %v", err)
		}
		fmt.Printf("Uploaded failed deploy results to %s\n", rURI)
		return err
	}
	d.addCommonMetadata(res)

	fmt.Println("Uploading successful deploy results")
	rURI, err := clouddeploy.UploadDeployResult(ctx, d.gcsClient, d.req.OutputGCSPath, res)
	if err != nil {
		return fmt.Errorf("error uploading deploy results: %v", err)
	}
	fmt.Printf("Uploaded deploy results to %s\n", rURI)
	return nil

}

func (d *deployer) deploy(ctx context.Context) (*clouddeploy.DeployResult, error) {
	inManifestGCSURI := d.req.ManifestGCSPath
	localManifest := "manifest.yaml"
	fmt.Printf("Downloading deploy input manifest from %q.\n", inManifestGCSURI)
	if _, err := clouddeploy.DownloadGCS(ctx, d.gcsClient, inManifestGCSURI, localManifest); err != nil {
		return nil, fmt.Errorf("unable to download manifest from %q: %v", inManifestGCSURI, err)
	}

	manifestData, err := d.applyModel(ctx, localManifest)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy model: %v", err)
	}

	manifestGCSURI := fmt.Sprintf("%s/%s", d.req.OutputGCSPath, "manifest.yaml")
	if err := clouddeploy.UploadGCS(ctx, d.gcsClient, manifestGCSURI, manifestData); err != nil {
		return nil, fmt.Errorf("error uploading manifest to GCS: %v", err)
	}

	return &clouddeploy.DeployResult{
		ResultStatus:  clouddeploy.DeploySucceeded,
		ArtifactFiles: []string{manifestGCSURI},
	}, nil
}

func (d *deployer) addCommonMetadata(rs *clouddeploy.DeployResult) {
	if rs.Metadata == nil {
		rs.Metadata = map[string]string{}
	}
	rs.Metadata[clouddeploy.CustomTargetSourceMetadataKey] = aiDeployerSampleName
	rs.Metadata[clouddeploy.CustomTargetCommitSha] = clouddeploy.GitCommit
}

func deployModelFromManifest(path string) (*aiplatform.GoogleCloudAiplatformV1DeployModelRequest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading manifest file: %v", err)
	}

	deployModelRequest := &aiplatform.GoogleCloudAiplatformV1DeployModelRequest{}

	if err = yaml.Unmarshal(data, deployModelRequest); err != nil {
		return nil, fmt.Errorf("unable to parse deploy model deployModelRequest from manifest file: %v", err)
	}

	return deployModelRequest, nil
}

func (d *deployer) applyModel(ctx context.Context, localManifest string) ([]byte, error) {

	deployModelRequest, err := deployModelFromManifest(localManifest)
	if err != nil {
		return nil, err
	}

	endpointRegion, err := fetchRegionFromEndpoint(d.params.endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse region from endpoint resource name: %v", err)
	}

	aiplatformService, err := newService(ctx, endpointRegion)
	if err != nil {
		return nil, fmt.Errorf("unable to create service: %v", err)
	}

	if d.req.Percentage != 100 {
		previousModel, err := determinePreviousModel(aiplatformService, d.params.endpoint, deployModelRequest.DeployedModel.Model)
		if err != nil {
			return nil, fmt.Errorf("unable to get previous model to canary against: %v", err)
		}
		previousPercentage, ok := deployModelRequest.TrafficSplit["previous-model"]
		if !ok {
			return nil, fmt.Errorf("expected input manifest trafficSplit stanza to have a 'previous-model' entry but did not find it")
		}
		delete(deployModelRequest.TrafficSplit, "previous-model")
		deployModelRequest.TrafficSplit[previousModel] = previousPercentage
	}
	op, err := aiplatformService.Projects.Locations.Endpoints.DeployModel(d.params.endpoint, deployModelRequest).Do()

	if err != nil {
		return nil, fmt.Errorf("unable to deploy model: %v", err)
	}

	if err := Poll(ctx, aiplatformService, op); err != nil {
		return nil, err
	}

	endpoint, err := aiplatformService.Projects.Locations.Endpoints.Get(d.params.endpoint).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to fetch endpoint: %v", err)
	}

	var modelsToUndeploy = map[string]bool{}
	for _, dm := range endpoint.DeployedModels {
		modelsToUndeploy[dm.Id] = true
	}

	for id, _ := range endpoint.TrafficSplit {
		delete(modelsToUndeploy, id)
	}

	undeployedCount := 0
	err = nil
	var lros []*aiplatform.GoogleLongrunningOperation
	for id, _ := range modelsToUndeploy {

		undeployRequest := &aiplatform.GoogleCloudAiplatformV1UndeployModelRequest{DeployedModelId: id}
		lro, lroErr := aiplatformService.Projects.Locations.Endpoints.UndeployModel(d.params.endpoint, undeployRequest).Do()
		if err != nil {
			fmt.Printf("error undeploying model: %v\n", err)
			err = lroErr
			undeployedCount += 1
		} else {
			lros = append(lros, lro)
		}
	}

	for pollErr := range pollChan(ctx, aiplatformService, lros...) {
		if pollErr != nil {
			fmt.Printf("Error in undeploy model operation: %v", err)
			err = pollErr
		}
	}

	return yaml.Marshal(deployModelRequest)
}

func determinePreviousModel(service *aiplatform.Service, endpointName, model string) (string, error) {
	endpoint, err := service.Projects.Locations.Endpoints.Get(endpointName).Do()
	if err != nil {
		return "", fmt.Errorf("unable to fetch endpoint: %v", err)
	}

	deployedModels := map[string]*aiplatform.GoogleCloudAiplatformV1DeployedModel{}

	for _, dm := range endpoint.DeployedModels {
		modelNameWithVersion := resolveDeployedModelNameWithVersion(dm)
		deployedModels[modelNameWithVersion] = dm
	}

	delete(deployedModels, model)

	if len(deployedModels) != 1 {
		return "", fmt.Errorf("unable to resolve previous deployed model to canary against. Not including the current model to be deployed, the endpoint has %d deployed models but expected only one", len(deployedModels))
	}

	firstModel := []*aiplatform.GoogleCloudAiplatformV1DeployedModel{}

	for _, dm := range deployedModels {
		firstModel = append(firstModel, dm)
	}
	return firstModel[0].Id, nil
}

func resolveDeployedModelNameWithVersion(deployedModel *aiplatform.GoogleCloudAiplatformV1DeployedModel) string {
	if strings.Contains(deployedModel.Model, "@") {
		return deployedModel.Model
	}
	return fmt.Sprintf("%s@%s", deployedModel.Model, deployedModel.ModelVersionId)
}

const (
	// wait for 30 seconds for a response from CCFE regarding an operation.
	lroOperationTimeout = 30 * time.Second
	// Polling duration, regardless of how long the lease is, we're going to poll for at most 30 mins.
	pollingTimeout = 30 * time.Minute
)

// Poll will return the status of an operation if it finished within "operationTimeout" or an error
// indicating that the operation is incomplete.
func Poll(ctx context.Context, service *aiplatform.Service, op *aiplatform.GoogleLongrunningOperation) error {

	opService := aiplatform.NewProjectsLocationsOperationsService(service)

	_, err := opService.Get(op.Name).Do()

	if err != nil {
		return fmt.Errorf("unable to get operation")
	}

	pollFunc := GetWaitFunc(opService, op.Name, ctx)

	err = wait.PollUntilContextTimeout(ctx, lroOperationTimeout, pollingTimeout, true, pollFunc)

	if err != nil {
		return err
	}
	return nil
}

// GetWaitFunc waits for stuff
func GetWaitFunc(service *aiplatform.ProjectsLocationsOperationsService, name string, ctx context.Context) wait.ConditionWithContextFunc {
	return func(ctx context.Context) (done bool, err error) {

		op, err := service.Get(name).Do()

		if err != nil {
			return false, err
		}

		if op.Done {
			return true, nil
		}

		return false, nil

	}
}

func pollChan(ctx context.Context, service *aiplatform.Service, lros ...*aiplatform.GoogleLongrunningOperation) <-chan error {
	var wg sync.WaitGroup
	out := make(chan error)
	wg.Add(len(lros))

	output := func(lro *aiplatform.GoogleLongrunningOperation) {
		out <- Poll(ctx, service, lro)
		wg.Done()
	}

	for _, lro := range lros {
		go output(lro)
	}

	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}
