package main

import (
	"context"
	"google.golang.org/api/aiplatform/v1"
	"testing"
	// "google.golang.org/api/aiplatform/v1"
	// "github.com/google/go-cmp/cmp"
)

// Tests that pipelineRequestFromManifest fails when given an incorrect path. Does not test correct path or incomplete file!
func TestPipelineRequestFromManifest(t *testing.T) {
	_, err := pipelineRequestFromManifest("")
	if err == nil {
		t.Errorf("Expected: error, Actual: %s", err)
	}

	_, err = pipelineRequestFromManifest("testPath")
	if err == nil {
		t.Errorf("Expected: error, Actual: %s", err)
	}

	_, err = pipelineRequestFromManifest(" ")
	if err == nil {
		t.Errorf("Expected: error, Actual: %s", err)
	}
}

// Tests that pipelineRequestFromManifest acts as expected when given a valid path.
// func TestPipelineRequestFromManifestSuccess(t *testing.T) {
// 	cont, err := pipelineRequestFromManifest("")
// 	if err == nil {
// 		t.Errorf("Expected: error, Actual: %s", err)
// 	}

// 	cont, err = pipelineRequestFromManifest("/usr/local/google/home/scortabarria/Desktop/cloud-deploy-samples/custom-targets/vertex-ai-pipeline/configuration/test.text")
// 	if err == nil {
// 		t.Errorf("Expected: error, Actual: %s", err)
// 	}

// 	cont, err = pipelineRequestFromManifest("/usr/local/google/home/scortabarria/Desktop/cloud-deploy-samples/custom-targets/vertex-ai-pipeline/configuration/test.yaml")
// 	if err != nil || cont.Parent == "" || cont.PipelineJobId == "" {
// 		t.Errorf("Expected: success, Actual: %s", err)
// 	}
// }

// Tests that deployPipeline fails as expected. Does not test actual deployment
func TestDeployPipeline(t *testing.T) {
	aiService, _ := newAIPlatformService(context.Background(), "us-central1")
	err := deployPipeline(context.Background(), aiService, "projects/scortabarria-internship/locations/us-central1", &aiplatform.GoogleCloudAiplatformV1CreatePipelineJobRequest{})
	if err == nil {
		t.Errorf("Expected: error, Actual: %s", err)
	}

	// job := &aiplatform.GoogleCloudAiplatformV1CreatePipelineJobRequest{}

	// err = deployPipeline(context.Background(), aiService, "", &aiplatform.GoogleCloudAiplatformV1CreatePipelineJobRequest{})
	// if err != nil{
	// 	t.Errorf("Expected: error, Actual: %s", err)
	// }
	// _ = aiService.Projects.Locations.PipelineJobs.Cancel("projects/scortabarria-internship/locations/us-central1", &aiplatform.GoogleCloudAiplatformV1CancelPipelineJobRequest{})
}
