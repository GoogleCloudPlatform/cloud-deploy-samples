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
)

const (
	// The name of the Helm deployer sample, this is passed back to Cloud Deploy
	// as metadata in the render and deploy results.
	helmDeployerSampleName = "clouddeploy-helm-sample"
)

func main() {
	if err := do(); err != nil {
		fmt.Fprintf(os.Stderr, "err: %v\n", err)
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
func createRequestHandler(ctx context.Context, cloudDeployRequest any, params *params, gcsClient *storage.Client) (requestHandler, error) {
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
		return nil, fmt.Errorf("received unsupported cloud deploy request type: %q", os.Getenv(clouddeploy.RequestTypeEnvKey))
	}
}
