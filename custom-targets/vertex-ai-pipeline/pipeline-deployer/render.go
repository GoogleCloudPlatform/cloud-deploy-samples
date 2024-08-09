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
	"github.com/GoogleCloudPlatform/cloud-deploy-samples/custom-targets/util/applysetters"
	"github.com/GoogleCloudPlatform/cloud-deploy-samples/custom-targets/util/clouddeploy"
	"google.golang.org/api/aiplatform/v1"
	"sigs.k8s.io/yaml"
)

const (
	// The default place to look for a pipelineJob configuration file if a specific location is not specified
	defaultConfigPath = "/workspace/source/pipelineJob.yaml"
	// Path to use when downloading the source input archive file.
	srcArchivePath = "/workspace/archive.tgz"
	// Path to use when unarchiving the source input.
	srcPath = "/workspace/source"
)

// renderer implements the handler interface for performing a render.
type renderer struct {
	gcsClient         *storage.Client
	aiPlatformService *aiplatform.Service
	params            *params
	req               *clouddeploy.RenderRequest
}

// process processes the Render params by generating the YAML representation of a
// CreatePipelineJobRequest object.
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

	out, err := r.renderCreatePipelineRequest()
	if err != nil {
		return nil, fmt.Errorf("error rendering createPipelineJobRequest params: %v", err)
	}

	fmt.Printf("Uploading deployed pipeline manifest.\n")

	mURI, err := r.req.UploadArtifact(ctx, r.gcsClient, "manifest.yaml", &clouddeploy.GCSUploadContent{Data: out})
	if err != nil {
		return nil, fmt.Errorf("error uploading createPipelineJobRequest manifest: %v", err)
	}

	fmt.Printf("Uploaded createPipelineJobRequest manifest to %s\n", mURI)

	return &clouddeploy.RenderResult{
		ResultStatus: clouddeploy.RenderSucceeded,
		ManifestFile: mURI,
	}, nil
}

// renderCreatePipelineRequest generates a CreatePipelineJobRequest object and returns its definition as a yaml-formatted string
func (r *renderer) renderCreatePipelineRequest() ([]byte, error) {
	if err := applyDeployParams(r.params.configPath); err != nil {
		return nil, fmt.Errorf("cannot apply deploy parameters to configuration file: %v", err)
	}

	configuration, err := loadConfigurationFile(r.params.configPath)
	if err != nil {
		return nil, fmt.Errorf("unable to obtain configuration data: %v", err)
	}

	// blank pipelineJob template
	pipelineJob := &aiplatform.GoogleCloudAiplatformV1PipelineJob{}

	if err = yaml.Unmarshal(configuration, pipelineJob); err != nil {
		return nil, fmt.Errorf("unable to parse configuration data into pipelineJob object: %v", err)
	}
	paramValues := r.params.pipelineParams

	if pipelineJob.TemplateUri == "" {
		pipelineJob.TemplateUri = r.params.pipeline
	}

	if pipelineJob.DisplayName == "" {
		pipelineJob.DisplayName = paramValues["model_display_name"]
	}

	paramValues["project_id"] = r.params.project
	paramString, err := json.Marshal(paramValues)
	if err != nil {
		fmt.Printf("Error marshalling JSON: %s", err)
		return nil, fmt.Errorf("unable to marshal params json")
	}
	pipelineJob.RuntimeConfig.ParameterValues = paramString

	request := &aiplatform.GoogleCloudAiplatformV1CreatePipelineJobRequest{PipelineJob: pipelineJob}
	return yaml.Marshal(request)
}

// addCommonMetadata inserts metadata into the render result that should be present
// regardless of render success or failure.
func (r *renderer) addCommonMetadata(rs *clouddeploy.RenderResult) {
	if rs.Metadata == nil {
		rs.Metadata = map[string]string{}
	}
	rs.Metadata[clouddeploy.CustomTargetSourceMetadataKey] = aiDeployerSampleName
	rs.Metadata[clouddeploy.CustomTargetSourceSHAMetadataKey] = clouddeploy.GitCommit
}

// applyDeployParams replaces templated parameters in the pipelineJob manifest with
// the actual values derived from deploy parameters.
func applyDeployParams(configPath string) error {
	fullPath, _ := determineConfigFileLocation(configPath)
	deployParams := clouddeploy.FetchDeployParameters()
	return applysetters.ApplyParams(fullPath, deployParams)
}

// determineConfigFileLocation determines where to look for the `pipelineJob.yaml`
// configuration file. Since this file is optional, we shouldn't necessarily err
// if the file is missing. However, if the configRelativePath is provided it means
// that the user specified this value as a deploy-parameter and we should check
// that we can open and read the file or fail the render if we cannot.
func determineConfigFileLocation(configRelativePath string) (string, bool) {
	configPath := defaultConfigPath
	shouldErrOnMissingFile := false

	if configRelativePath != "" {
		configPath = fmt.Sprintf("%s/%s", srcPath, configRelativePath)
		shouldErrOnMissingFile = true
	}

	return configPath, shouldErrOnMissingFile
}

// loadConfigurationFile loads and returns the configuration file for the target if it exists.
func loadConfigurationFile(configPath string) ([]byte, error) {
	filePath, shouldErrOnMissingFile := determineConfigFileLocation(configPath)
	fmt.Errorf("HERE: %s", filePath)
	fileInfo, err := os.Stat(filePath)
	if err != nil && shouldErrOnMissingFile {
		return nil, err
	}

	if fileInfo != nil {
		return os.ReadFile(filePath)
	}
	return nil, nil
}
