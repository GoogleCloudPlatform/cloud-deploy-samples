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

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/cloud-deploy-samples/custom-targets/util/clouddeploy"
	"github.com/GoogleCloudPlatform/cloud-deploy-samples/packages/cdenv"
	"google.golang.org/api/aiplatform/v1"
)

// Environment variable keys specific to the vertex ai deployer. These are provided via
// deploy parameters in Cloud Deploy.
const (
	pipelineEnvKey = "CLOUD_DEPLOY_customTarget_vertexAIPipeline"
	configPathKey  = "CLOUD_DEPLOY_customTarget_vertexAIPipelineJobConfiguration"
	paramValsKey   = "CLOUD_DEPLOY_customTarget_vertexAIPipelineJobParameterValues"
	locValsKey     = "CLOUD_DEPLOY_customTarget_location"
	projectValsKey = "CLOUD_DEPLOY_customTarget_projectID"
)

// requestHandler interface provides methods for handling the Cloud Deploy params.
type requestHandler interface {
	// Process processes the Cloud Deploy params.
	process(ctx context.Context) error
}

// createRequestHandler creates a requestHandler for the provided Cloud Deploy request.
func createRequestHandler(cloudDeployRequest any, params *params, gcsClient *storage.Client, service *aiplatform.Service) (requestHandler, error) {
	switch r := cloudDeployRequest.(type) {
	case *clouddeploy.RenderRequest:
		return &renderer{
			req:               r,
			params:            params,
			gcsClient:         gcsClient,
			aiPlatformService: service,
		}, nil

	case *clouddeploy.DeployRequest:
		return &deployer{
			req:               r,
			params:            params,
			gcsClient:         gcsClient,
			aiPlatformService: service,
		}, nil

	default:
		return nil, fmt.Errorf("received unsupported cloud deploy request type: %q", os.Getenv(cdenv.RequestTypeEnvKey))
	}
}

// params contains the deploy parameter values passed into the execution environment.
type params struct {
	// The Project ID for the target environment.
	project string

	// The location where the ML pipeline will be deployed
	location string

	// The pipeline template that is being deployed
	pipeline string

	// The directory path where the renderer should look for target-specific configuration
	// for this deployment, if not provided the renderer will check for a pipelineJob.yaml
	// file in the root working directory.
	configPath string

	// Pipeline parameters obtained via deploy parameters. Hold parameters necessary
	// for the createPipelineJobRequest, such as the prompt dataset
	pipelineParams map[string]string
}

// determineParams returns the supported params provided in the execution environment via environment variables.
func determineParams() (*params, error) {
	location, found := os.LookupEnv(locValsKey)
	if !found {
		return nil, fmt.Errorf("environment variable %s not found", locValsKey)
	}
	if location == "" {
		return nil, fmt.Errorf("environment variable %s contains empty string", locValsKey)
	}

	project, found := os.LookupEnv(projectValsKey)
	if !found {
		return nil, fmt.Errorf("required environment variable %s not found", projectValsKey)
	}
	if project == "" {
		return nil, fmt.Errorf("environment variable %s contains empty string", projectValsKey)
	}

	pipeline, found := os.LookupEnv(pipelineEnvKey)
	if !found {
		return nil, fmt.Errorf("required environment variable %s not found", pipelineEnvKey)
	}
	if pipeline == "" {
		return nil, fmt.Errorf("environment variable %s contains empty string", pipelineEnvKey)
	}

	paramString, found := os.LookupEnv(paramValsKey)
	if !found {
		return nil, fmt.Errorf("required environment variable %s not found", paramValsKey)
	}
	var pipelineParams map[string]string
	err := json.Unmarshal([]byte(paramString), &pipelineParams)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal params json")
	}

	if len(pipelineParams) == 0 {
		return nil, fmt.Errorf("environment variable %s contains empty string", paramValsKey)
	}

	config, found := os.LookupEnv(configPathKey)
	if !found {
		return nil, fmt.Errorf("required environment variable %s not found", configPathKey)
	}
	if config == "" {
		return nil, fmt.Errorf("environment variable %s contains empty string", configPathKey)
	}

	return &params{
		project:        project,
		pipeline:       pipeline,
		configPath:     config,
		location:       location,
		pipelineParams: pipelineParams,
	}, nil
}
