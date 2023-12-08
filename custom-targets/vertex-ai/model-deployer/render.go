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
	"github.com/GoogleCloudPlatform/cloud-deploy-samples/custom-targets/util/applysetters"
	"github.com/GoogleCloudPlatform/cloud-deploy-samples/custom-targets/util/clouddeploy"

	"google.golang.org/api/aiplatform/v1"
	"os"
	"regexp"
	"strings"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
	"sigs.k8s.io/yaml"
)

const (
	// The default place to look for a deployed model configuration file if a specific location is not specified
	defaultConfigPath = "/workspace/source/deployedModel.yaml"

	// Path to use when downloading the source input archive file.
	srcArchivePath = "/workspace/archive.tgz"
	// Path to use when unarchiving the source input.
	srcPath = "/workspace/source"
)

var (
	modelRegex    = regexp.MustCompile("^projects/([^/]+)/locations/([^/]+)/models/([^/]+)$")
	endpointRegex = regexp.MustCompile("^projects/([^/]+)/locations/([^/]+)/endpoints/([^/]+)$")
)

// renderer implements the handler interface for performing a render.
type renderer struct {
	gcsClient *storage.Client
	params    *params
	req       *clouddeploy.RenderRequest
}

// process processes the Render params by generating the YAML representation of a
// DeployModel params, optionally
// the output GCS path to be used by the deploy params.
func (r *renderer) process(ctx context.Context) error {
	fmt.Println("Processing render request")
	res, err := r.render(ctx)
	if err != nil {
		fmt.Printf("Render failed: %v\n", err)
		res := &clouddeploy.RenderResult{
			ResultStatus:   clouddeploy.RenderFailed,
			FailureMessage: err.Error(),
		}
		r.addCommonMetadata(res)
		fmt.Println("Uploading failed render results")
		rURI, err := r.req.UploadResult(ctx, r.gcsClient, res)
		if err != nil {
			return fmt.Errorf("error uploading failed render results: %v", err)
		}
		fmt.Printf("Uploaded failed render results to %s\n", rURI)
		return err
	}
	r.addCommonMetadata(res)

	fmt.Println("Uploading successful render results")
	rURI, err := r.req.UploadResult(ctx, r.gcsClient, res)
	if err != nil {
		return fmt.Errorf("error uploading render results: %v", err)
	}
	fmt.Printf("Uploaded render results to %s\n", rURI)
	return nil
}

func (r *renderer) render(ctx context.Context) (*clouddeploy.RenderResult, error) {
	fmt.Printf("Downloading render input archive to %s and unarchiving to %s\n", srcArchivePath, srcPath)
	inURI, err := r.req.DownloadAndUnarchiveInput(ctx, r.gcsClient, srcArchivePath, srcPath)
	if err != nil {
		return nil, fmt.Errorf("unable to download and unarchive render input: %v", err)
	}
	fmt.Printf("Downloaded render input archive from %s\n", inURI)

	out, err := renderDeployModelRequest(ctx, r.params.configPath, r.params.endpoint, r.params.model, r.params.minReplicaCount, int64(r.req.Percentage))
	if err != nil {
		return nil, fmt.Errorf("error rendering deploy model params: %v", err)
	}

	fmt.Printf("Uploading deployed model manifest.\n")

	mURI, err := r.req.UploadArtifact(ctx, r.gcsClient, "manifest.yaml", &clouddeploy.GCSUploadContent{Data: out})
	if err != nil {
		return nil, fmt.Errorf("error uploading deployed model manifest: %v", err)
	}

	fmt.Printf("Uploaded deployed model manifest to %s\n", mURI)

	return &clouddeploy.RenderResult{
		ResultStatus: clouddeploy.RenderSucceeded,
		ManifestFile: mURI,
	}, nil
}

func (r *renderer) addCommonMetadata(rs *clouddeploy.RenderResult) {
	if rs.Metadata == nil {
		rs.Metadata = map[string]string{}
	}
	rs.Metadata[clouddeploy.CustomTargetSourceMetadataKey] = aiDeployerSampleName
	rs.Metadata[clouddeploy.CustomTargetSourceSHAMetadataKey] = clouddeploy.GitCommit
}

func validateRequest(modelNameFromDeployParameter, endpointName string, minReplicaCountParameter int64, deployedModel *aiplatform.GoogleCloudAiplatformV1DeployedModel) error {
	modelRegion, err := fetchRegionFromModel(modelNameFromDeployParameter)
	if err != nil {
		return fmt.Errorf("unable to parse region from model: %v", err)
	}

	endpointRegion, err := fetchRegionFromEndpoint(endpointName)
	if err != nil {
		return fmt.Errorf("unable to parse region from endpoint: %v", err)
	}

	if endpointRegion != modelRegion {
		return fmt.Errorf("The model to be deployed must be in the same region as the endpoint. Copy the model to the region the  endpoint is located, or make an endpoint in the same region as the model")
	}

	if err = verifyModelNameNotDefinedInConfig(deployedModel); err != nil {
		return err
	}

	if err = verifyMinReplicaCountHasNoConflicts(deployedModel, minReplicaCountParameter); err != nil {
		return err
	}

	return nil
}

func verifyMinReplicaCountHasNoConflicts(deployedModel *aiplatform.GoogleCloudAiplatformV1DeployedModel, deployParameterValue int64) error {

	configValue := minReplicaCountFromConfig(deployedModel)

	// checks if minReplicaCount is not defined either in deploy parameter or config file
	if configValue == deployParameterValue {
		if configValue == 0 {
			return fmt.Errorf("minReplicaCount must either be defined in the config file or provided to the render operation through a deploy parameter using 'vertexAIMinReplicaCount' key")
		}
	}

	// only other valid format is if either but not both are 0
	if configValue == 0 || deployParameterValue == 0 {
		return nil
	}
	return fmt.Errorf("the minReplicaCount parameter is defined in both the provided config file and as a deploy parameter and both values differ from each other, please define minReplicaCount in the config file or as a deploy-parameter")
}

func minReplicaCountFromConfig(deployedModel *aiplatform.GoogleCloudAiplatformV1DeployedModel) int64 {
	if deployedModel.DedicatedResources != nil {
		return deployedModel.DedicatedResources.MinReplicaCount
	}
	return 0
}
func verifyModelNameNotDefinedInConfig(deployedModel *aiplatform.GoogleCloudAiplatformV1DeployedModel) error {

	if deployedModel.Model != "" {
		return fmt.Errorf("model to deployed must be supplied as a deploy parameter and not in the config file")
	}

	if deployedModel.ModelVersionId != "" {
		return fmt.Errorf("the model version id to deploy must be supplied as part of the vertexAIModel deployparamater containing the model to be deployed")
	}

	return nil

}

// renderDeployModelRequest generates a model definition
func renderDeployModelRequest(ctx context.Context, configPath, endpointName, modelName string, minReplicaCount int64, percentage int64) ([]byte, error) {

	if err := applyDeployParams(configPath); err != nil {
		return nil, fmt.Errorf("cannot apply deploy parameters to configuration file: %v", err)
	}

	modelRegion, err := fetchRegionFromModel(modelName)
	if err != nil {
		return nil, fmt.Errorf("unable to parse region from model name: %v", err)
	}
	aiplatformService, err := newService(ctx, modelRegion)
	if err != nil {
		return nil, fmt.Errorf("unable to create aiplatform service: %v", err)
	}

	// blank deployed model template
	deployedModel := &aiplatform.GoogleCloudAiplatformV1DeployedModel{}

	configuration, err := loadConfigurationFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("unable to obtain configuration data: %v", err)
	}

	if err = yaml.Unmarshal(configuration, deployedModel); err != nil {
		return nil, fmt.Errorf("unable to parse configuration data into DeployModel object: %v", err)
	}

	model, err := fetchModel(aiplatformService, modelName)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch model: %v", err)
	}

	modelNameWithVersionId := resolveModelWithVersion(model)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve model version: %v", err)
	}

	if err := validateRequest(modelNameWithVersionId, endpointName, minReplicaCount, deployedModel); err != nil {
		return nil, fmt.Errorf("manifest validation failed: %v", err)
	}
	deployedModel.Model = modelNameWithVersionId

	if deployedModel.DedicatedResources == nil {
		deployedModel.DedicatedResources = &aiplatform.GoogleCloudAiplatformV1DedicatedResources{MinReplicaCount: minReplicaCount}
	}

	if deployedModel.DedicatedResources.MinReplicaCount == 0 {
		deployedModel.DedicatedResources.MinReplicaCount = minReplicaCount
	}

	// deploy model params requires this field to be non-nil. Setting to the default "n1-standard-2"
	// if it's not already set
	if deployedModel.DedicatedResources.MachineSpec == nil {
		deployedModel.DedicatedResources.MachineSpec = &aiplatform.GoogleCloudAiplatformV1MachineSpec{MachineType: "n1-standard-2"}
	}

	if deployedModel.DedicatedResources.MachineSpec.MachineType == "" {
		deployedModel.DedicatedResources.MachineSpec.MachineType = "n1-standard-2"
	}

	trafficSplit := map[string]int64{}
	trafficSplit["0"] = percentage

	if percentage != 100 {
		trafficSplit["previous-model"] = 100 - percentage
	}

	request := &aiplatform.GoogleCloudAiplatformV1DeployModelRequest{DeployedModel: deployedModel, TrafficSplit: trafficSplit}

	return yaml.Marshal(request)
}

func applyDeployParams(configPath string) error {
	fullPath, _ := determineConfigFileLocation(configPath)
	deployParams := clouddeploy.FetchDeployParameters()
	return applysetters.ApplyParams(fullPath, deployParams)
}

func resolveModelWithVersion(model *aiplatform.GoogleCloudAiplatformV1Model) string {
	if strings.Contains(model.Name, "@") {
		return model.Name
	}
	return fmt.Sprintf("%s@%s", model.Name, model.VersionId)
}

// extracts the region from the model region name
func fetchRegionFromModel(modelName string) (string, error) {
	matches := modelRegex.FindStringSubmatch(modelName)
	if len(matches) == 0 {
		return "", fmt.Errorf("unable to parse model name")
	}

	return matches[2], nil
}

// extracts the region from the endpoint resource name
func fetchRegionFromEndpoint(endpointName string) (string, error) {
	matches := endpointRegex.FindStringSubmatch(endpointName)
	if len(matches) == 0 {
		return "", fmt.Errorf("unable to parse endpoint name")
	}

	return matches[2], nil
}

// newService generates a Service that can make API calls in the specified region
func newService(ctx context.Context, region string) (*aiplatform.Service, error) {
	endPointOption := option.WithEndpoint(fmt.Sprintf("%s-aiplatform.googleapis.com", region))
	regionalService, err := aiplatform.NewService(ctx, endPointOption)
	if err != nil {
		return nil, fmt.Errorf("unable to authenticate")
	}

	return regionalService, nil
}

func determineConfigFileLocation(configRelativePath string) (string, bool) {

	configPath := defaultConfigPath
	shouldErrOnMissingFile := false

	if configRelativePath != "" {
		configPath = fmt.Sprintf("%s/%s/deployedModel.yaml", srcPath, configRelativePath)
		shouldErrOnMissingFile = true
	}

	return configPath, shouldErrOnMissingFile

}

// loadConfigurationFile loads and returns the configuration file for the target if it exists.
func loadConfigurationFile(configPath string) ([]byte, error) {
	filePath, shouldErrOnMissingFile := determineConfigFileLocation(configPath)

	fileInfo, err := os.Stat(filePath)
	if err != nil && shouldErrOnMissingFile {
		return nil, err
	}

	if fileInfo != nil {
		return os.ReadFile(filePath)
	}
	return nil, nil
}

func fetchModel(service *aiplatform.Service, modelName string) (*aiplatform.GoogleCloudAiplatformV1Model, error) {
	model, err := service.Projects.Locations.Models.Get(modelName).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %v", err)
	}
	return model, nil
}
