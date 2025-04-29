package main

import (
	"context"
	"fmt"
	"google3/third_party/cloud_deploy_samples/custom_targets/util/clouddeploy/clouddeploy"
	"google3/third_party/golang/google_api/aiplatform/v1/aiplatform"
	"os"
	"strconv"
	"strings"

	"google3/third_party/golang/cloud_google_com/go/storage/v/v1/storage"
)

// Environment variable keys specific to the vertex ai deployer. These are provided via
// deploy parameters in Cloud Deploy.
const (
	minReplicaCountEnvKey = "CLOUD_DEPLOY_customTarget_vertexAIMinReplicaCount"
	modelEnvKey           = "CLOUD_DEPLOY_customTarget_vertexAIModel"
	endpointEnvKey        = "CLOUD_DEPLOY_customTarget_vertexAIEndpoint"
	aliasEnvKey           = "CLOUD_DEPLOY_customTarget_vertexAIAliases"
	configPathKey         = "CLOUD_DEPLOY_customTarget_vertexAIConfigurationPath"
)

// deploy parameters that the custom target requires to be present and provided during render and deploy operations.
const (
	modelDPKey    = "customTarget/vertexAIModel"
	endpointDPKey = "customTarget/vertexAIEndpoint"
	aliasDPKey    = "customTarget/vertexAIAliases"
)

var addAliasesMode bool

// requestHandler interface provides methods for handling the Cloud Deploy params.
type requestHandler interface {
	// Process processes the Cloud Deploy params.
	process(ctx context.Context) error
}

// createRequestHandler creates a requestHandler for the provided Cloud Deploy request.
func createRequestHandler(cloudDeployRequest interface{}, params *params, gcsClient *storage.Client, service *aiplatform.Service) (requestHandler, error) {

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
	// The minimum replica count for the deployed model obtained via a deploy parameter
	minReplicaCount int64

	// The model to be deployed. May or may not contain a tag or version number.
	// format is "projects/{project}/locations/{location}/models/{modelId}[@versionId|alias].
	model string

	// The endpoint where the model will be deployed
	// format is "projects/{project}/locations/{location}/endpoints/{endpointId}.
	endpoint string

	// directory path where the renderer should look for target-specific configuration
	// for this deployment, if not provided the renderer will check for a deployModel.yaml
	// fie in the root working directory.
	configPath string
}

// determineParams returns the supported params provided in the execution environment via environment variables.
func determineParams() (*params, error) {

	replicaCount, err := strconv.Atoi(os.Getenv(minReplicaCountEnvKey))
	if err != nil {
		replicaCount = 0
	}

	model, found := os.LookupEnv(modelEnvKey)
	if !found {
		fmt.Printf("Required environment variable %s not found. This variable is derived from deploy parameter: %s, please verify that a valid Vertex AI model resource name was provided through this deploy parameter.\n", modelEnvKey, modelDPKey)
		return nil, fmt.Errorf("required environment variable %s not found", modelEnvKey)
	}
	if model == "" {
		fmt.Printf("environment variable %s is empty. This variable is derived from deploy parameter: %s, please verify that a valid Vertex AI model resource name was provided through this deploy parameter.\n", modelEnvKey, modelDPKey)
		return nil, fmt.Errorf("environment variable %s contains empty string", modelEnvKey)
	}

	endpoint, found := os.LookupEnv(endpointEnvKey)
	if !found {
		fmt.Printf("Required environment variable %s not found. This variable is derived from deploy parameter: %s, please verify that a valid Vertex AI model resource name was provided through this deploy parameter.\n", endpointEnvKey, endpointDPKey)
		return nil, fmt.Errorf("required environment variable %s not found", modelEnvKey)
	}
	if model == "" {
		fmt.Printf("environment variable %s is empty. This variable is derived from deploy parameter: %s, please verify that a valid Vertex AI model resource name was provided through this deploy parameter.\n", endpointEnvKey, endpointDPKey)
		return nil, fmt.Errorf("environment variable %s contains empty string", modelEnvKey)
	}

	return &params{
		model:           model,
		endpoint:        endpoint,
		minReplicaCount: int64(replicaCount),
		configPath:      os.Getenv(configPathKey),
	}, nil
}

// addAliasesRequest contains information needed to assign aliases to a model during a post deploy hook
type addAliasesRequest struct {
	// new aliases to apply to the model
	aliases []string

	// Cloud Deploy project
	project string
	// Cloud Deploy location.
	location string
	// Cloud Deploy target
	target string
	// Cloud Deploy delivery pipeline.
	pipeline string
	// Cloud Deploy release.
	release string
	// phase
	phase string
}

// newAliasHandler returns a handler for processing alias assignment requests.
func newAliasHandler(gcsClient *storage.Client) (requestHandler, error) {

	aliasParameter := os.Getenv(aliasEnvKey)
	if len(aliasParameter) == 0 {
		return nil, fmt.Errorf("when 'add aliases mode' is enabled', at least one alias needs to be passed to the custom action through %s deploy parameter", aliasDPKey)
	}

	aliases := strings.Split(aliasParameter, ",")

	request := &addAliasesRequest{
		project:  os.Getenv(clouddeploy.ProjectEnvKey),
		location: os.Getenv(clouddeploy.LocationEnvKey),
		pipeline: os.Getenv(clouddeploy.PipelineEnvKey),
		release:  os.Getenv(clouddeploy.ReleaseEnvKey),
		target:   os.Getenv(clouddeploy.TargetEnvKey),
		phase:    os.Getenv(clouddeploy.PhaseEnvKey),
		aliases:  aliases,
	}
	return &aliasAssigner{gcsClient: gcsClient, request: request}, nil
}
