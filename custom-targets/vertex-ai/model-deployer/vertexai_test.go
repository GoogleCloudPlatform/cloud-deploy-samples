package main

import (
	"github.com/google/go-cmp/cmp"
	"google.golang.org/api/aiplatform/v1"
	"testing"
)

// Tests that deployModelFromManifest fails when given an incorrect path. Does not test correct path or incomplete file!
func TestDeployModelFromManifestFails(t *testing.T) {
	_, err := deployModelFromManifest("")
	if err == nil {
		t.Errorf("Expected: error, Actual: %s", err)
	}

	_, err = deployModelFromManifest("testPath")
	if err == nil {
		t.Errorf("Expected: error, Actual: %s", err)
	}
}

// Tests that regionFromModel fails when give an empty string or an invalid model path
func TestRegionFromModelFail(t *testing.T) {
	_, err := regionFromModel("")
	if err == nil {
		t.Errorf("Expected: error, Actual: %s", err)
	}

	_, err = regionFromModel("not a path")
	if err == nil {
		t.Errorf("Expected: error, Actual: %s", err)
	}

	_, err = regionFromModel("projects/scortabarria-internship/locations/test-location/")
	if err == nil {
		t.Errorf("Expected: error, Actual: %s", err)
	}

	_, err = regionFromModel("projects/scortabarria-internship/locations//models/test-model")
	if err == nil {
		t.Errorf("Expected: error, Actual: %s", err)
	}

	_, err = regionFromModel("projects/scortabarria-internship/locations/models/test-model")
	if err == nil {
		t.Errorf("Expected: error, Actual: %s", err)
	}
}

// Tests that the method regionFromModel works as intended when given a valid model
func TestRegionFromModelPass(t *testing.T) {
	loc, err := regionFromModel("projects/scortabarria-internship/locations/test-location/models/test-model")
	if d := cmp.Diff(loc, "test-location"); d != "" || err != nil {
		t.Errorf("ERROR: %s", err)
	}
}

// Tests that regionFromEndpoint fails when give an empty string or an invalid endpoint path
func TestRegionFromEndpointFail(t *testing.T) {
	_, err := regionFromEndpoint("")
	if err == nil {
		t.Errorf("Expected: error, Actual: %s", err)
	}

	_, err = regionFromEndpoint("not a path")
	if err == nil {
		t.Errorf("Expected: error, Actual: %s", err)
	}

	_, err = regionFromEndpoint("projects/scortabarria-internship/locations/test-location/")
	if err == nil {
		t.Errorf("Expected: error, Actual: %s", err)
	}

	_, err = regionFromEndpoint("projects/scortabarria-internship/locations//endpoints/test-endpoint")
	if err == nil {
		t.Errorf("Expected: error, Actual: %s", err)
	}

	_, err = regionFromEndpoint("projects/scortabarria-internship/locations//endpoints/test-endpoint")
	if err == nil {
		t.Errorf("Expected: error, Actual: %s", err)
	}
}

// Tests that the regionFromEndpoint successfully returns location from endpoint
func TestRegionFromEndpointPass(t *testing.T) {
	loc, err := regionFromEndpoint("projects/scortabarria-internship/locations/test-location/endpoints/test-endpoint")
	if diff := cmp.Diff(loc, "test-location"); diff != "" || err != nil {
		t.Errorf("ERROR: %s", err)
	}

}

// Tests that minReplicaCoundFromConfig returns 0 when no minReplicaCount is specified in the configuration
// file. If it is specified, it returns that value
func TestMinReplicaCountFromConfig(t *testing.T) {
	deployedModel := &aiplatform.GoogleCloudAiplatformV1DeployedModel{}
	if num := minReplicaCountFromConfig(deployedModel); num != 0 {
		t.Errorf("Error: num was expected to be 0, Actual: %v", num)
	}

	deployedModel.DedicatedResources = &aiplatform.GoogleCloudAiplatformV1DedicatedResources{
		MinReplicaCount: 5,
	}
	if num := minReplicaCountFromConfig(deployedModel); num != 5 {
		t.Errorf("Error: num was expected to be 5, Actual %v", num)
	}
}
