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
	"os"
	"path"

	config "cloud.google.com/go/config/apiv1"
	"cloud.google.com/go/config/apiv1/configpb"
	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/cloud-deploy-samples/custom-targets/util/clouddeploy"
	"github.com/ghodss/yaml"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	// Key to use for the deployment name in the metadata results when deploy succeeds.
	deploymentMetadataKey = "deployment"
	// Key to use for the revision name in the metadata results when deploy succeeds.
	revisionMetadataKey = "revision"
)

// deployer implements the requestHandler interface for deploy requests.
type deployer struct {
	req       *clouddeploy.DeployRequest
	params    *params
	imClient  *config.Client
	gcsClient *storage.Client
}

// process processes a deploy request and uploads succeeded or failed results to GCS for Cloud Deploy.
func (d *deployer) process(ctx context.Context) error {
	fmt.Println("Processing deploy request")

	res, err := d.deploy(ctx)
	if err != nil {
		fmt.Printf("Deploy failed: %v\n", err)
		dr := &clouddeploy.DeployResult{
			ResultStatus:   clouddeploy.DeployFailed,
			FailureMessage: err.Error(),
			Metadata: map[string]string{
				clouddeploy.CustomTargetSourceMetadataKey:    imDeployerSampleName,
				clouddeploy.CustomTargetSourceSHAMetadataKey: clouddeploy.GitCommit,
			},
		}
		fmt.Println("Uploading failed deploy results")
		rURI, err := d.req.UploadResult(ctx, d.gcsClient, dr)
		if err != nil {
			return fmt.Errorf("error uploading failed deploy results: %v", err)
		}
		fmt.Printf("Uploaded failed deploy results to %s\n", rURI)
		return err
	}

	fmt.Println("Uploading deploy results")
	rURI, err := d.req.UploadResult(ctx, d.gcsClient, res)
	if err != nil {
		return fmt.Errorf("error uploading deploy results: %v", err)
	}
	fmt.Printf("Uploaded deploy results to %s\n", rURI)
	return nil
}

// deploy performs the following steps:
//  1. Create or update the Infrastructure Manager Deployment based on the Deployment YAML created at render time.
//
// Returns either the deploy results or an error if the deploy failed.
func (d *deployer) deploy(ctx context.Context) (*clouddeploy.DeployResult, error) {
	renderedDeploymentPath := path.Join(srcPath, renderedDeploymentFileName)
	fmt.Printf("Downloading rendered Deployment to %s\n", renderedDeploymentPath)
	dURI, err := d.req.DownloadInput(ctx, d.gcsClient, renderedDeploymentFileName, renderedDeploymentPath)
	if err != nil {
		return nil, fmt.Errorf("unable to download rendered deployment with object suffix %s: %v", renderedDeploymentFileName, err)
	}
	fmt.Printf("Downloaded rendered Deployment from %s\n", dURI)
	rd, err := renderedDeployment(renderedDeploymentPath)
	if err != nil {
		return nil, fmt.Errorf("error parsing rendered deployment: %v", err)
	}
	deployment, err := d.applyDeployment(ctx, rd)
	if err != nil {
		return nil, err
	}
	revName := deployment.LatestRevision
	fmt.Printf("Created latest Revision %s\n", revName)

	// Ensure the Deployment reached a terminal state after creating/updating it. If for some reason it's still in
	// progress then we poll it until it reaches a terminal state. The polling logic checks whether the latest revision
	// changes in case the Deployment is updated outside the context of this deployer.
	if isInProgressDeployment(deployment.State) {
		fmt.Printf("Polling Deployment %s until a terminal state is reached, current state: %s\n", deployment.Name, deployment.State.String())
		var err error
		deployment, err = pollDeploymentUntilTerminal(ctx, d.imClient, deployment.Name, revName)
		if err != nil {
			return nil, err
		}
		fmt.Printf("Finished polling Deployment %s until terminal state, current state: %s\n", deployment.Name, deployment.State.String())
	}

	fmt.Printf("Retrieving Revision %s\n", revName)
	rev, err := getRevision(ctx, d.imClient, revName)
	if err != nil {
		return nil, fmt.Errorf("error getting revision %s: %v", revName, err)
	}
	fmt.Printf("Revision %s executed in Cloud Build %s\n", revName, rev.Build)

	if isSucceededDeployment(deployment.State) {
		fmt.Printf("Deployment Succeeded with latest Revision %s\n", revName)
		return processDeploymentSucceeded(ctx, deployment, rev)
	}
	fmt.Printf("Deployment Failed with latest Revision %s\n", revName)
	return nil, processDeploymentFailed(ctx, deployment, rev)
}

// renderedDeployment returns the Infrastructure Manager Deployment created at render time that is defined
// in YAML format at the provided path.
func renderedDeployment(deploymentYAMLPath string) (*configpb.Deployment, error) {
	b, err := os.ReadFile(deploymentYAMLPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %v", deploymentYAMLPath, err)
	}
	j, err := yaml.YAMLToJSON(b)
	if err != nil {
		return nil, fmt.Errorf("error converting deployment yaml to json: %v", err)
	}
	deployment := &configpb.Deployment{}
	if err := protojson.Unmarshal(j, deployment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal deployment: %v", err)
	}
	return deployment, nil
}

// applyDeployment either creates or updates an existing Infrastructure Manager Deployment with the
// provided Deployment configuration.
func (d *deployer) applyDeployment(ctx context.Context, renderedDeployment *configpb.Deployment) (*configpb.Deployment, error) {
	deploymentName := renderedDeployment.Name
	fmt.Printf("Checking whether Deployment %s exists\n", deploymentName)
	if _, err := getDeployment(ctx, d.imClient, deploymentName); status.Code(err) == codes.NotFound {
		// Deployment doesn't exist yet.
		fmt.Printf("Creating Deployment %s\n", deploymentName)
		d, err := createDeployment(ctx, d.imClient, renderedDeployment)
		if err != nil {
			return nil, fmt.Errorf("error creating deployment %s: %v", deploymentName, err)
		}
		fmt.Printf("Created Deployment %s, current state: %s\n", deploymentName, d.State.String())
		return d, nil

	} else if err != nil {
		return nil, fmt.Errorf("error getting deployment %s: %v", deploymentName, err)
	}

	// Deployment already exists so it needs to be updated.
	fmt.Printf("Updating Deployment %s\n", deploymentName)
	postD, err := updateDeployment(ctx, d.imClient, renderedDeployment)
	if err != nil {
		return nil, fmt.Errorf("error updating deployment %s: %v", deploymentName, err)
	}
	fmt.Printf("Updated Deployment %s, current state: %s\n", deploymentName, postD.State.String())
	return postD, nil
}

// processDeploymentSucceeded handles a successful Deployment and returns a successful deploy result that includes the
// Infrastructure Manager revision's outputs in the result metadata.
func processDeploymentSucceeded(ctx context.Context, deployment *configpb.Deployment, rev *configpb.Revision) (*clouddeploy.DeployResult, error) {
	metadata := map[string]string{
		clouddeploy.CustomTargetSourceMetadataKey:    imDeployerSampleName,
		clouddeploy.CustomTargetSourceSHAMetadataKey: clouddeploy.GitCommit,
		deploymentMetadataKey:                        deployment.Name,
		revisionMetadataKey:                          rev.Name,
	}
	for k, v := range rev.ApplyResults.Outputs {
		mv, err := v.Value.MarshalJSON()
		if err != nil {
			return nil, fmt.Errorf("unable to marshal revision output %s", k)
		}
		metadata[k] = string(mv)
	}
	res := &clouddeploy.DeployResult{
		ResultStatus: clouddeploy.DeploySucceeded,
		Metadata:     metadata,
	}
	return res, nil
}

// processDeploymentFailed handles a failed Deployment by logging various information from the Infrastructure Manager
// resources to provide context on the failure.
func processDeploymentFailed(ctx context.Context, deployment *configpb.Deployment, rev *configpb.Revision) error {
	failureMessage := fmt.Sprintf("Deployment %s had state %s at failure time.", deployment.Name, deployment.State.String())
	// If there is an error code present then include it in the failure message for Cloud Deploy.
	if deployment.ErrorCode != configpb.Deployment_ERROR_CODE_UNSPECIFIED {
		failureMessage = fmt.Sprintf("%s Error code: %s", failureMessage, deployment.ErrorCode)
	}
	fmt.Printf("%s\n", failureMessage)

	fmt.Printf("Revision state: %s, error code: %s\n", rev.State, rev.ErrorCode)
	fmt.Printf("Revision state details: %s\n", rev.StateDetail)
	for i, tfe := range rev.TfErrors {
		if len(tfe.ErrorDescription) != 0 {
			fmt.Printf("Revision Terraform error %d: %v\n", i+1, tfe.ErrorDescription)
		}
	}
	return fmt.Errorf("An error occurred: %s", failureMessage)
}
