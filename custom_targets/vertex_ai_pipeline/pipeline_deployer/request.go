package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"google3/third_party/cloud_deploy_samples/custom_targets/util/clouddeploy/clouddeploy"
	"google3/third_party/golang/cloud_google_com/go/storage/v/v1/storage"
	"google3/third_party/golang/google_api/aiplatform/v1/aiplatform"
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
		return nil, fmt.Errorf("received unsupported cloud deploy request type: %q", os.Getenv(clouddeploy.RequestTypeEnvKey))
	}
}

// params contains the deploy parameter values passed into the execution environment.
type params struct {
	// The Project ID for the target environment.
	project string

	// The location where the ML pipeline will be deployed
	location string

	//T he pipeline template that is being deployed
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
