package main

import (
	"context"
	"strings"
	"testing"

	"google3/third_party/cloud_deploy_samples/custom_targets/util/clouddeploy/clouddeploy"
	"google3/third_party/golang/cloud_google_com/go/storage/v/v1/storage"
)

// Tests that render works as expected. Does not test valid renderer.
func TestRender(t *testing.T) {
	gcsClient, _ := storage.NewClient(context.Background())
	newRenderer := &renderer{
		params:    &params{},
		gcsClient: gcsClient,
		req:       &clouddeploy.RenderRequest{},
	}
	_, err := newRenderer.render(context.Background())
	if in := strings.Contains(err.Error(), "unable to download and unarchive render input"); !in {
		t.Errorf("Expected: unable to download and unarchive render input, Received: %s", err)
	}
}

// Tests that renderDeployModelRequest() handles error from empty renderer. Does not test valid renderer!
func TestRenderCreatePipelineRequest(t *testing.T) {
	newRenderer := &renderer{
		params: &params{},
	}
	_, err := newRenderer.renderCreatePipelineRequest()
	if in := strings.Contains(err.Error(), "cannot apply deploy parameters to configuration file"); !in {
		t.Errorf("Expected: cannot apply deploy parameters to configuration file, Received: %s", err)
	}

	newRenderer.params.configPath = "configuration/test.yaml"
	_, err = newRenderer.renderCreatePipelineRequest()
	if in := strings.Contains(err.Error(), "cannot apply deploy parameters to configuration file"); !in {
		t.Errorf("Expected: cannot apply deploy parameters to configuration file, Received: %s", err)
	}

}

// Tests that addCommonMetadata populates the RenderResult as expected
func TestRendAddCommonMetadata(t *testing.T) {
	newRenderer := &renderer{}
	rendResult := &clouddeploy.RenderResult{}
	if myMap := rendResult.Metadata; myMap != nil {
		t.Errorf("Expected empty field, received: %s", myMap)
	}
	newRenderer.addCommonMetadata(rendResult)
	if _, exists := rendResult.Metadata[clouddeploy.CustomTargetSourceMetadataKey]; !exists {
		t.Errorf("Error: map missing %s key", clouddeploy.CustomTargetSourceMetadataKey)
	}
	if _, exists := rendResult.Metadata[clouddeploy.CustomTargetSourceSHAMetadataKey]; !exists {
		t.Errorf("Error: map missing %s key", clouddeploy.CustomTargetSourceSHAMetadataKey)
	}
}

// Tests that applyDeployParams fails when given an invalid path. Does not test valid path!
func TestApplyDeployParamsFails(t *testing.T) {
	err := applyDeployParams("")
	if err == nil {
		t.Errorf("Expected: error, Actual: %s", err)
	}

	err = applyDeployParams("not a path")
	if err == nil {
		t.Errorf("Expected: error, Actual: %s", err)
	}
}

// Tests that determineConfigLocation fails when an invalid path is passed in but passes when no path is
// given. This is due to the fact that the path is optional.
func TestDetermineConfigLocation(t *testing.T) {
	path, shouldErr := determineConfigFileLocation("")
	if shouldErr != false {
		t.Errorf("Expected shouldErr to be false, Actual: %t", shouldErr)
	}
	if path != "/workspace/source/pipelineJob.yaml" {
		t.Errorf("Expected path to be /workspace/source/pipelineJob.yaml, received: %s", path)
	}

	path, shouldErr = determineConfigFileLocation(" ")
	if shouldErr != true {
		t.Errorf("Expected shouldErr to be true, received: %t", shouldErr)
	}
	if path != "/workspace/source/ " {
		t.Errorf("Expected path to be /workspace/source/ , received: %s", path)
	}

	path, shouldErr = determineConfigFileLocation("testPath")
	if shouldErr != true {
		t.Errorf("Expected shouldErr to be true, received: %t", shouldErr)
	}
	if path != "/workspace/source/testPath" {
		t.Errorf("Expected path to be /workspace/source/testPath, received: %s", path)
	}
}

// Tests that loadConfigurationFile acts as expected when a path or an empty string is passed in. Does not test valid path!
func TestLoadConfigurationFile(t *testing.T) {
	content, err := loadConfigurationFile("")
	if err != nil || content != nil {
		t.Errorf("Expected: nil and nil, received: %s and %s", err, content)
	}

	content, err = loadConfigurationFile(" ")
	if content != nil || err == nil {
		t.Errorf("Expected: nil and error, received: %s and %s", content, err)
	}

	content, err = loadConfigurationFile("not a path")
	if content != nil || err == nil {
		t.Errorf("Expected: nil and error, received: %s and %s", content, err)
	}

}
