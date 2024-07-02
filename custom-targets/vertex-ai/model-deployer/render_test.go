package main

import (
	"testing"
	"github.com/GoogleCloudPlatform/cloud-deploy-samples/custom-targets/util/clouddeploy"
	"google.golang.org/api/aiplatform/v1"
)

//Tests that renderDeployModelRequest() handles error from empty renderer. Does not test valid renderer!
func TestRenderDeployModelRequest(t *testing.T) {
	params := &params{}
	newRenderer := &renderer{
		params:            params,
	}
	if _, err := newRenderer.renderDeployModelRequest(); err == nil{
		t.Errorf("Error expected, received: %s", err)
	}
}

//Tests that addCommonMetadata populates the RenderResult as expected
func TestAddCommonMetadata(t *testing.T) {
	newRenderer := &renderer{}
	rendResult := &clouddeploy.RenderResult{}
	if myMap := rendResult.Metadata; myMap != nil{
		t.Errorf("Expected empty field, received: %s", myMap)
	}
	newRenderer.addCommonMetadata(rendResult)
	if _, exists := rendResult.Metadata[clouddeploy.CustomTargetSourceMetadataKey]; !exists{
		t.Errorf("Error: map missing %s key", clouddeploy.CustomTargetSourceMetadataKey)
	}
	if _, exists := rendResult.Metadata[clouddeploy.CustomTargetSourceSHAMetadataKey]; !exists{
		t.Errorf("Error: map missing %s key", clouddeploy.CustomTargetSourceSHAMetadataKey)
	}
}

//Tests that applyDeployParams fails when given an invalid path. Does not test valid path!
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

//Tests that determineConfigLocation fails when an invalid path is passed in but passes when no path is 
//given. This is due to the fact that the path is optional. Does not test valid path!
func TestDetermineConfigLocation(t *testing.T) {
	path, shouldErr := determineConfigFileLocation("")
	if shouldErr != false {
		t.Errorf("Expected shouldErr to be false, Actual: %t", shouldErr)
	}
	if path != "/workspace/source/deployedModel.yaml"{
		t.Errorf("Expected path to be /workspace/source/deployedModel.yaml, received: %s", path)
	}

	path, shouldErr = determineConfigFileLocation(" ")
	if shouldErr != true {
		t.Errorf("Expected shouldErr to be true, received: %t", shouldErr)
	}
	if path != "/workspace/source/ "{
		t.Errorf("Expected path to be /workspace/source/ , received: %s", path)
	}

	path, shouldErr = determineConfigFileLocation("testPath")
	if shouldErr != true {
		t.Errorf("Expected shouldErr to be true, received: %t", shouldErr)
	}
	if path != "/workspace/source/testPath"{
		t.Errorf("Expected path to be /workspace/source/testPath, received: %s", path)
	}
}

//Tests that loadConfigurationFile acts as expected when a path or an empty string is passed in. Does not test valid path!
func TestLoadConfigurationFile(t *testing.T) {
	content, err := loadConfigurationFile("")
	if err != nil  || content != nil{
		t.Errorf("Expected: nil and nil, received: %s and %s", err, content)
	}

	content, err = loadConfigurationFile(" ")
	if content != nil || err == nil{
		t.Errorf("Expected: nil and error, received: %s and %s", content, err)
	}

	content, err = loadConfigurationFile("not a path")
	if content != nil || err == nil{
		t.Errorf("Expected: nil and error, received: %s and %s", content, err)
	}
}

//This test goes through all of the possible errors in validateRequest. It then tests if validateRequest 
//succeeds.
func TestValidateRequest(t *testing.T) {
	deployedModel := &aiplatform.GoogleCloudAiplatformV1DeployedModel{}
	err := validateRequest("", "", int64(0), deployedModel)
	if err == nil{
		t.Errorf("Expected: error from invalid model, Received: %s", err)
	}

	path := "projects/scortabarria-internship/locations/test-location1/models/test-model"
	err = validateRequest(path, "", int64(0), deployedModel)
	if err == nil{
		t.Errorf("Expected: error from invalid endpointName, Received: %s", err)
	}

	endpoint := "projects/scortabarria-internship/locations/test-location2/endpoints/test-endpoint"
	err = validateRequest(path, endpoint, int64(0), deployedModel)
	if err == nil{
		t.Errorf("Expected: error from conflicting regions, Received: %s", err)
	}

	endpoint = "projects/scortabarria-internship/locations/test-location1/endpoints/test-endpoint"
	deployedModel.Model = "testName"
	err = validateRequest(path, endpoint, int64(0), deployedModel)
	if err == nil{
		t.Errorf("Expected: error from model name in config, Received: %s", err)
	}

	deployedModel.Model = ""
	deployedModel.DedicatedResources = &aiplatform.GoogleCloudAiplatformV1DedicatedResources{
		MinReplicaCount: 5,
	}
	err = validateRequest(path, endpoint, int64(2), deployedModel)
	if err == nil{
		t.Errorf("Expected: error from conflicting minReplicaCount, Received: %s", err)
	}

	err = validateRequest(path, endpoint, int64(0), deployedModel)
	if err != nil{
		t.Errorf("ERROR: %s", err)
	}

}


//This test verifies that minReplicaCount is defined somewhere. We test how verifyMinReplicaHasNoConflicts
//handles it being defined nowhere, in one place and in conflicting places
func TestVerifyMinReplicaHasNoConflicts(t *testing.T) {
	deployedModel := &aiplatform.GoogleCloudAiplatformV1DeployedModel{}
	paramVal := int64(0)
	if err := verifyMinReplicaCountHasNoConflicts(deployedModel, paramVal); err == nil{
		t.Errorf("Expected: error, received: %s", err)
	}

	deployedModel.DedicatedResources = &aiplatform.GoogleCloudAiplatformV1DedicatedResources{
		MinReplicaCount: 5,
	}
	if err := verifyMinReplicaCountHasNoConflicts(deployedModel, paramVal); err != nil{
		t.Errorf("ERROR: %s", err)
	}

	paramVal = int64(2)
	if err := verifyMinReplicaCountHasNoConflicts(deployedModel, paramVal); err == nil{
		t.Errorf("Expected: error, received: %s", err)
	}

	paramVal = int64(5)
	if err := verifyMinReplicaCountHasNoConflicts(deployedModel, paramVal); err != nil{
		t.Errorf("ERROR: %s", err)
	}

	deployedModel.DedicatedResources = &aiplatform.GoogleCloudAiplatformV1DedicatedResources{
		MinReplicaCount: 0,
	}
	if err := verifyMinReplicaCountHasNoConflicts(deployedModel, paramVal); err != nil{
		t.Errorf("ERROR: %s", err)
	}
}



//This test checks that verifyModelNameNotDefinedInConfig works as expected. The model name and versionId must
//not be defined in the config for the function to pass
func TestVerifyModelNameNotDefinedInConfig(t *testing.T) {
	deployedModel := &aiplatform.GoogleCloudAiplatformV1DeployedModel{}
	err := verifyModelNameNotDefinedInConfig(deployedModel)
	if err != nil{
		t.Errorf("ERROR: %s", err)
	}

	deployedModel.Model = "testName"
	if err := verifyModelNameNotDefinedInConfig(deployedModel); err == nil{
		t.Errorf("Expected: error, Received: %v", err)
	}

	deployedModel.ModelVersionId = "12"
	if err := verifyModelNameNotDefinedInConfig(deployedModel); err == nil{
		t.Errorf("Expected: error, Received: %v", err)
	}

	deployedModel.Model = ""
	if err := verifyModelNameNotDefinedInConfig(deployedModel); err == nil{
		t.Errorf("Expected: error, Received: %v", err)
	}	
}