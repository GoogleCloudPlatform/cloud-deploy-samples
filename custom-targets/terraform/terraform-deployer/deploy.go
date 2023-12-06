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
	"encoding/json"
	"fmt"
	"os"
	"path"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/cloud-deploy-samples/custom-targets/util/clouddeploy"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/mholt/archiver/v3"
)

// deployer implements the requestHandler interface for deploy requests.
type deployer struct {
	req       *clouddeploy.DeployRequest
	params    *params
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
				clouddeploy.CustomTargetSourceMetadataKey: tfDeployerSampleName,
				"custom-target-source-commit-sha":         clouddeploy.GitCommit,
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
//  1. Initialize the Terraform configuration only to install providers. Modules and backend were initialized at render time.
//  2. Apply the Terraform configuration.
//  3. Get the Terraform state and upload to GCS as a deploy artifact.
//
// Returns either the deploy results or an error if the deploy failed.
func (d *deployer) deploy(ctx context.Context) (*clouddeploy.DeployResult, error) {
	// Download the Terraform configuration uploaded at render time and unarchive it in the same
	// directory that was used at render time.
	fmt.Printf("Downloading Terraform configuration archive to %s\n", srcArchivePath)
	inURI, err := d.req.DownloadInput(ctx, d.gcsClient, renderedArchiveName, srcArchivePath)
	if err != nil {
		return nil, fmt.Errorf("unable to download deploy input with object suffix %s: %v", renderedArchiveName, err)
	}
	fmt.Printf("Downloaded Terraform configuration archive from %s\n", inURI)

	archiveFile, err := os.Open(srcArchivePath)
	if err != nil {
		return nil, fmt.Errorf("unable to open archive file %s: %v", srcArchivePath, err)
	}
	fmt.Printf("Unarchiving Terraform configuration in %s to %s\n", srcArchivePath, srcPath)
	if err := archiver.NewTarGz().Unarchive(archiveFile.Name(), srcPath); err != nil {
		return nil, fmt.Errorf("unable to unarchive terraform configuration: %v", err)
	}

	terraformConfigPath := path.Join(srcPath, d.params.configPath)
	fmt.Println("Initializing Terraform configuration to install providers")
	if _, err := terraformInit(terraformConfigPath, &terraformInitOptions{disableBackendInitialization: true, disableModuleDownloads: true}); err != nil {
		return nil, fmt.Errorf("error running terraform init to install providers: %v", err)
	}
	if _, err := terraformApply(terraformConfigPath, &terraformApplyOptions{applyParallelism: d.params.applyParallelism, lockTimeout: d.params.lockTimeout}); err != nil {
		return nil, fmt.Errorf("error running terraform apply: %v", err)
	}
	fmt.Println("Finished applying Terraform configuration")

	fmt.Println("Getting the Terraform state to provide as a deploy artifact")
	ts, err := terraformShowState(terraformConfigPath)
	if err != nil {
		return nil, fmt.Errorf("error getting terraform state after apply: %v", err)
	}
	fmt.Println("Extracting Terraform output values from the Terraform state")
	metadata, err := extractOutputsFromTfState(ts)
	if err != nil {
		return nil, fmt.Errorf("error extracting terraform outputs from the terraform state: %v", err)
	}
	fmt.Println("Uploading Terraform state as a deploy artifact")
	stateGCSURI, err := d.req.UploadArtifact(ctx, d.gcsClient, "deployed-state.json", &clouddeploy.GCSUploadContent{Data: ts})
	if err != nil {
		return nil, fmt.Errorf("error uploading terraform state deploy artifact: %v", err)
	}
	fmt.Printf("Uploaded Terraform state deploy artifact to %s\n", stateGCSURI)

	// Metadata consists of the Terraform output values and an indicator that the deploy was handled by the
	// cloud deploy terraform sample.
	metadata[clouddeploy.CustomTargetSourceMetadataKey] = tfDeployerSampleName
	metadata["custom-target-source-commit-sha"] = clouddeploy.GitCommit

	deployResult := &clouddeploy.DeployResult{
		ResultStatus:  clouddeploy.DeploySucceeded,
		ArtifactFiles: []string{stateGCSURI},
		Metadata:      metadata,
	}
	return deployResult, nil
}

// extractOutputsFromTfState returns a map of the Terraform outputs in the provided JSON Terraform state. The map
// values are the JSON strings of the output values.
func extractOutputsFromTfState(jsonTfState []byte) (map[string]string, error) {
	s := &tfjson.State{}
	if err := s.UnmarshalJSON(jsonTfState); err != nil {
		return nil, fmt.Errorf("unable to unmarshal terraform state: %v", err)
	}

	res := make(map[string]string)
	// Parse each Terraform output from the Terraform state into JSON strings.
	for k, v := range s.Values.Outputs {
		sv, err := json.Marshal(v.Value)
		if err != nil {
			return nil, fmt.Errorf("unable to marshal terraform state output for key %s: %v", k, err)
		}
		res[k] = string(sv)
	}
	return res, nil
}
