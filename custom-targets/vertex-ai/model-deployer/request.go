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
	"flag"
	"fmt"
	"github.com/GoogleCloudPlatform/cloud-deploy-samples/custom-targets/util/clouddeploy"
	"os"
	"strconv"
	"strings"

	"cloud.google.com/go/storage"
)

// Environment variable keys specific to the vertex ai deployer. These are provided via
// deploy parameters in Cloud Deploy.
const (
	minReplicaCountEnvKey = "CLOUD_DEPLOY_customTarget_vertexAIMinReplicaCount"
	modelEnvKey           = "CLOUD_DEPLOY_customTarget_vertexAIModel"

	endpointEnvKey = "CLOUD_DEPLOY_customTarget_vertexAIEndpoint"

	aliasEnvKey = "CLOUD_DEPLOY_customTarget_vertexAIAliases"

	configPathKey = "CLOUD_DEPLOY_customTarget_vertexAIConfigurationPath"
)

var addAliasesMode bool

// requestHandler interface provides methods for handling the Cloud Deploy params.
type requestHandler interface {
	// Process processes the Cloud Deploy params.
	process(ctx context.Context) error
}

// determineRequestHandler reads the environment variables set by Cloud Deploy in order to determine
// the type of params that needs to be processed and returns the appropriate handler. Returns an error
// if the params cannot be handled.
func determineRequestHandler(ctx context.Context) (requestHandler, error) {
	flag.BoolVar(&addAliasesMode, "add-aliases-mode", false, "if enabled, adds aliases set in vertexAIAliases environment variable to the deployed model")
	flag.Parse()
	gcsClient, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to create gcs client: %v", err)
	}

	if addAliasesMode {
		return newAliasHandler(gcsClient)
	}

	reqType := os.Getenv(clouddeploy.RequestTypeEnvKey)
	switch reqType {
	case "RENDER":
		return newRenderHandler(gcsClient)
	case "DEPLOY":
		return newDeployHandler(gcsClient)
	default:
		return nil, fmt.Errorf("received unexpected Cloud Deploy params type: %v", reqType)
	}
}

// newRenderHandler returns a handler for processing render requests.
func newRenderHandler(gcsClient *storage.Client) (requestHandler, error) {
	percentage, err := strconv.Atoi(os.Getenv(clouddeploy.PercentageEnvKey))
	if err != nil {
		return nil, fmt.Errorf("failed to parse env var %q", clouddeploy.PercentageEnvKey)
	}

	params, err := determineParams()

	if err != nil {
		return nil, fmt.Errorf("unable to parse params: %v", err)
	}

	rr := &clouddeploy.RenderRequest{
		Target:        os.Getenv(clouddeploy.TargetEnvKey),
		Percentage:    percentage,
		InputGCSPath:  os.Getenv(clouddeploy.InputGCSEnvKey),
		OutputGCSPath: os.Getenv(clouddeploy.OutputGCSEnvKey),
	}

	return &renderer{
		gcsClient: gcsClient,
		req:       rr,
		params:    params}, nil
}

// newDeployHandler returns a handler for processing deploy requests.
func newDeployHandler(gcsClient *storage.Client) (requestHandler, error) {
	percentage, err := strconv.Atoi(os.Getenv(clouddeploy.PercentageEnvKey))
	if err != nil {
		return nil, fmt.Errorf("failed to parse env var %q: %v", clouddeploy.PercentageEnvKey, err)
	}

	params, err := determineParams()

	if err != nil {
		return nil, fmt.Errorf("unable to parse params: %v", err)
	}

	request := &clouddeploy.DeployRequest{
		Percentage:      percentage,
		ManifestGCSPath: os.Getenv(clouddeploy.ManifestGCSEnvKey),
		OutputGCSPath:   os.Getenv(clouddeploy.OutputGCSEnvKey),
	}
	return &deployer{gcsClient: gcsClient, req: request, params: params}, nil
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
		fmt.Println("unable to obtain minReplicaCount parameter from environment variable, will attempt to check if in the configuration file if its provided")
		replicaCount = 0
	}

	return &params{
		minReplicaCount: int64(replicaCount),
		model:           os.Getenv(modelEnvKey),
		endpoint:        os.Getenv(endpointEnvKey),
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
		return nil, fmt.Errorf("when 'add aliases mode' is enabled', at least one alias needs to be passed to the custom action through %s deploy parameter", aliasEnvKey)
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
