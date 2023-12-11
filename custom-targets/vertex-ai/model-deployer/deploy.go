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

// deploy.go contains logic to deploy a model to a vertex AI endpoint.
package main

import (
	"context"
	"fmt"
	"github.com/GoogleCloudPlatform/cloud-deploy-samples/custom-targets/util/clouddeploy"
	"google.golang.org/api/aiplatform/v1"
	"sigs.k8s.io/yaml"

	"cloud.google.com/go/storage"
)

const aiDeployerSampleName = "clouddeploy-vertex-ai-sample"

const localManifest = "manifest.yaml"

// deployer implements the handler interface to deploy a model using the vertex AI API.
type deployer struct {
	gcsClient *storage.Client
	params    *params
	req       *clouddeploy.DeployRequest
}

// process processes the Deploy request, and performs the vertex AI model deployment.
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
		rURI, err := d.req.UploadResult(ctx, d.gcsClient, dr)
		if err != nil {
			return fmt.Errorf("error uploading failed deploy results: %v", err)
		}
		fmt.Printf("Uploaded failed deploy results to %s\n", rURI)
		return err
	}
	d.addCommonMetadata(res)

	fmt.Println("Uploading successful deploy results")
	rURI, err := d.req.UploadResult(ctx, d.gcsClient, res)
	if err != nil {
		return fmt.Errorf("error uploading deploy results: %v", err)
	}
	fmt.Printf("Uploaded deploy results to %s\n", rURI)
	return nil

}

// deploy performs the Vertex AI model deployment
func (d *deployer) deploy(ctx context.Context) (*clouddeploy.DeployResult, error) {

	if err := d.downloadManifest(ctx); err != nil {
		return nil, err
	}

	manifestData, err := d.applyModel(ctx, localManifest)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy model: %v", err)
	}

	mURI, err := d.req.UploadArtifact(ctx, d.gcsClient, "manifest.yaml", &clouddeploy.GCSUploadContent{Data: manifestData})
	if err != nil {
		return nil, fmt.Errorf("error uploading deploy artifact: %v", err)
	}

	return &clouddeploy.DeployResult{
		ResultStatus:  clouddeploy.DeploySucceeded,
		ArtifactFiles: []string{mURI},
	}, nil
}

// downloadManifest downloads the rendered manifest from Google Cloud Storage to the local manifest file path
func (d *deployer) downloadManifest(ctx context.Context) error {
	fmt.Printf("Downloading deploy input manifest from %q.\n", d.req.ManifestGCSPath)

	downloadPath, err := d.req.DownloadManifest(ctx, d.gcsClient, localManifest)
	if err != nil {
		fmt.Printf("Unable to download deployed manifest from: %s.\n", d.req.ManifestGCSPath)
		return fmt.Errorf("unable to download deploy input from %s: %v", d.req.ManifestGCSPath, err)
	}

	fmt.Printf("Downloaded deploy input manifest from: %s\n", downloadPath)

	return nil
}

// addCommonMetadata inserts metadata into the deploy result that should be present
// regardless of deploy success or failure.
func (d *deployer) addCommonMetadata(rs *clouddeploy.DeployResult) {
	if rs.Metadata == nil {
		rs.Metadata = map[string]string{}
	}
	rs.Metadata[clouddeploy.CustomTargetSourceMetadataKey] = aiDeployerSampleName
	rs.Metadata[clouddeploy.CustomTargetSourceSHAMetadataKey] = clouddeploy.GitCommit
}

// applyModel deploys the DeployModelRequest parsed from `localManifest`
// it returns the DeployedModelRequest object that was used in yaml format.
func (d *deployer) applyModel(ctx context.Context, localManifest string) ([]byte, error) {

	deployModelRequest, err := deployModelFromManifest(localManifest)
	if err != nil {
		return nil, fmt.Errorf("unable to load DeployModelRequest from manifest: %v", err)
	}

	endpointRegion, err := fetchRegionFromEndpoint(d.params.endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse region from endpoint resource name: %v", err)
	}

	aiplatformService, err := newAIPlatformService(ctx, endpointRegion)
	if err != nil {
		return nil, fmt.Errorf("unable to create AI Platform Service: %v", err)
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
