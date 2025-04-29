package main

import (
	"context"
	"fmt"
	"os"

	"google3/third_party/golang/google_api/aiplatform/v1/aiplatform"
	"google3/third_party/golang/google_api/option/option"
	"google3/third_party/golang/kubeyaml/yaml"
)

// pipelineRequestFromManifest loads the file provided in `path` and returns the parsed CreatePipelineJobRequest
// from the data.
func pipelineRequestFromManifest(path string) (*aiplatform.GoogleCloudAiplatformV1CreatePipelineJobRequest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading manifest file: %v", err)
	}

	createPipelineRequest := &aiplatform.GoogleCloudAiplatformV1CreatePipelineJobRequest{}
	if err = yaml.Unmarshal(data, createPipelineRequest); err != nil {
		return nil, fmt.Errorf("unable to parse createPipelineJobRequest from manifest file: %v", err)
	}

	return createPipelineRequest, nil
}

// newAIPlatformService generates a Service that can make API calls in the specified region.
func newAIPlatformService(ctx context.Context, region string) (*aiplatform.Service, error) {
	endPointOption := option.WithEndpoint(fmt.Sprintf("%s-aiplatform.googleapis.com", region))
	regionalService, err := aiplatform.NewService(ctx, endPointOption)
	if err != nil {
		return nil, fmt.Errorf("unable to authenticate")
	}
	return regionalService, nil
}

// deployPipeline performs the deployPipeline request and awaits the resulting operation until it completes, it times out or an error occurs.
func deployPipeline(ctx context.Context, aiPlatformService *aiplatform.Service, parent string, request *aiplatform.GoogleCloudAiplatformV1CreatePipelineJobRequest) error {
	fmt.Printf("PARENT: %s; REQUEST: %v", parent, request.PipelineJob)
	_, err := aiPlatformService.Projects.Locations.PipelineJobs.Create(parent, request.PipelineJob).Do()
	if err != nil {
		return fmt.Errorf("unable to deploy pipeline: %v", err)
	}
	return nil
}
