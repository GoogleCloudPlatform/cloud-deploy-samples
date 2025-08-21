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
	"github.com/GoogleCloudPlatform/cloud-deploy-samples/packages/gcs"
	"google.golang.org/api/aiplatform/v1"
	"sigs.k8s.io/yaml"

	"cloud.google.com/go/storage"
)

const aiDeployerSampleName = "clouddeploy-vertex-ai-sample"

const localManifest = "manifest.yaml"

// deployer implements the handler interface to deploy a model using the vertex AI API.
type deployer struct {
	gcsClient         *storage.Client
	aiPlatformService *aiplatform.Service
	params            *params
	req               *clouddeploy.DeployRequest
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
		d.addCommonMetadata(dr)
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

	mURI, err := d.req.UploadArtifact(ctx, d.gcsClient, "manifest.yaml", &gcs.UploadContent{Data: manifestData})
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

	if d.req.Percentage != 100 {
		if err := d.makeManifestChangesForCanary(deployModelRequest); err != nil {
			return nil, fmt.Errorf("unable to make canary changes to the manifest: %v", err)
		}
	}

	if err := deployModel(ctx, d.aiPlatformService, d.params.endpoint, deployModelRequest); err != nil {
		return nil, fmt.Errorf("unable to deploy model: %v", err)
	}

	if err := undeployNoTrafficModels(ctx, d.aiPlatformService, d.params.endpoint); err != nil {
		return nil, fmt.Errorf("unable to undeploy models from endpoint: %v", err)
	}

	return yaml.Marshal(deployModelRequest)
}

// makeManifestChangesForCanary generates a traffic split configuration such that traffic is routed to exactly two models:
// the new model being introduced, and the model that was previously deployed.
func (d *deployer) makeManifestChangesForCanary(deployModelRequest *aiplatform.GoogleCloudAiplatformV1DeployModelRequest) error {
	previousModel, err := fetchPreviousModel(d.aiPlatformService, d.params.endpoint, deployModelRequest.DeployedModel.Model)
	if err != nil {
		return fmt.Errorf("unable to get previous model to canary against: %v", err)
	}
	previousPercentage, ok := deployModelRequest.TrafficSplit["previous-model"]
	if !ok {
		return fmt.Errorf("expected input manifest trafficSplit stanza to have a 'previous-model' entry but did not find it")
	}
	delete(deployModelRequest.TrafficSplit, "previous-model")
	deployModelRequest.TrafficSplit[previousModel] = previousPercentage

	return nil
}
