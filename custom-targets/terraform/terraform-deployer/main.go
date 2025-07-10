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

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/cloud-deploy-samples/custom-targets/util/clouddeploy"
	"github.com/GoogleCloudPlatform/cloud-deploy-samples/packages/cdenv"
)

const (
	// The name of the Terraform deployer sample, this is passed back to Cloud Deploy
	// as metadata in the render and deploy results.
	tfDeployerSampleName = "clouddeploy-terraform-sample"
)

func main() {
	if err := do(); err != nil {
		fmt.Printf("err: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Done!")
}

func do() error {
	ctx := context.Background()
	gcsClient, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("unable to create cloud storage client: %v", err)
	}
	req, err := clouddeploy.DetermineRequest(ctx, gcsClient, []string{})
	if err != nil {
		return fmt.Errorf("unable to determine cloud deploy request: %v", err)
	}
	params, err := determineParams()
	if err != nil {
		return fmt.Errorf("unable to determine params: %v", err)
	}
	if err := setTerraformEnvVars(); err != nil {
		return err
	}
	h, err := createRequestHandler(ctx, req, params, gcsClient)
	if err != nil {
		return err
	}
	return h.process(ctx)
}

// requestHandler interface provides methods for handling the Cloud Deploy request.
type requestHandler interface {
	// Process processes the Cloud Deploy request.
	process(ctx context.Context) error
}

// createRequestHandler creates a requestHandler for the provided Cloud Deploy request.
func createRequestHandler(ctx context.Context, cloudDeployRequest interface{}, params *params, gcsClient *storage.Client) (requestHandler, error) {
	switch r := cloudDeployRequest.(type) {
	case *clouddeploy.RenderRequest:
		return &renderer{
			req:       r,
			params:    params,
			gcsClient: gcsClient,
		}, nil

	case *clouddeploy.DeployRequest:
		return &deployer{
			req:       r,
			params:    params,
			gcsClient: gcsClient,
		}, nil

	default:
		return nil, fmt.Errorf("received unsupported cloud deploy request type: %q", os.Getenv(cdenv.RequestTypeEnvKey))
	}
}

// setTerraformEnvVars sets environment variables consumed by Terraform that are required to execute
// the terraform commands.
func setTerraformEnvVars() error {
	// Setting "TF_IN_AUTOMATION" to any value will adjust terraform cli output to avoid suggesting
	// commands to run, which is only helpful when a human is executing the commands.
	if err := os.Setenv("TF_IN_AUTOMATION", "clouddeploy"); err != nil {
		return fmt.Errorf("unable to set TF_IN_AUTOMATION environment variable: %v", err)
	}
	// Setting "TF_INPUT" to false will behave as if `-input=false` is specified when running any
	// terraform commands. Since these commands are executing in a CD tool we do not want to
	// allow input prompts at runtime.
	if err := os.Setenv("TF_INPUT", "false"); err != nil {
		return fmt.Errorf("unable to set TF_INPUT environment variable: %v", err)
	}
	return nil
}
