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

	"cloud.google.com/go/storage"
	"google.golang.org/api/aiplatform/v1"
	"os"
	"regexp"
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

	out, err := r.renderDeployModelRequest(ctx)
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

// renderDeployModelRequest generates a model definition
func (r *renderer) renderDeployModelRequest(ctx context.Context) ([]byte, error) {

	if err := applyDeployParams(r.params.configPath); err != nil {
		return nil, fmt.Errorf("cannot apply deploy parameters to configuration file: %v", err)
	}

	modelRegion, err := fetchRegionFromModel(r.params.model)
	if err != nil {
		return nil, fmt.Errorf("unable to parse region from model name: %v", err)
	}
	aiplatformService, err := newAIPlatformService(ctx, modelRegion)
	if err != nil {
		return nil, fmt.Errorf("unable to create aiplatform service: %v", err)
	}

	// blank deployed model template
	deployedModel := &aiplatform.GoogleCloudAiplatformV1DeployedModel{}

	configuration, err := loadConfigurationFile(r.params.configPath)
	if err != nil {
		return nil, fmt.Errorf("unable to obtain configuration data: %v", err)
	}

	if err = yaml.Unmarshal(configuration, deployedModel); err != nil {
		return nil, fmt.Errorf("unable to parse configuration data into DeployModel object: %v", err)
	}

	model, err := fetchModel(aiplatformService, r.params.model)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch model: %v", err)
	}

	modelNameWithVersionId := resolveModelWithVersion(model)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve model version: %v", err)
	}

	if err := validateRequest(modelNameWithVersionId, r.params.endpoint, r.params.minReplicaCount, deployedModel); err != nil {
		return nil, fmt.Errorf("manifest validation failed: %v", err)
	}
	deployedModel.Model = modelNameWithVersionId

	if deployedModel.DedicatedResources == nil {
		deployedModel.DedicatedResources = &aiplatform.GoogleCloudAiplatformV1DedicatedResources{MinReplicaCount: r.params.minReplicaCount}
	}

	if deployedModel.DedicatedResources.MinReplicaCount == 0 {
		deployedModel.DedicatedResources.MinReplicaCount = r.params.minReplicaCount
	}

	// deploy model params requires this field to be non-nil. Setting to the default "n1-standard-2"
	// if it's not already set
	if deployedModel.DedicatedResources.MachineSpec == nil {
		deployedModel.DedicatedResources.MachineSpec = &aiplatform.GoogleCloudAiplatformV1MachineSpec{MachineType: "n1-standard-2"}
	}

	if deployedModel.DedicatedResources.MachineSpec.MachineType == "" {
		deployedModel.DedicatedResources.MachineSpec.MachineType = "n1-standard-2"
	}

	percentage := int64(r.req.Percentage)
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
