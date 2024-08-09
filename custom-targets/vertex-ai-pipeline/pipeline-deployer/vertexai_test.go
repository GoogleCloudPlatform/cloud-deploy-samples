package main

import (
	"context"
	"testing"

	"google.golang.org/api/aiplatform/v1"
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

// Tests that deployPipeline fails as expected. Does not test actual deployment
func TestDeployPipeline(t *testing.T) {
	aiService, _ := newAIPlatformService(context.Background(), "us-central1")
	err := deployPipeline(context.Background(), aiService, "projects/scortabarria-internship/locations/us-central1", &aiplatform.GoogleCloudAiplatformV1CreatePipelineJobRequest{})
	if err == nil {
		t.Errorf("Expected: error, Actual: %s", err)
	}
}
