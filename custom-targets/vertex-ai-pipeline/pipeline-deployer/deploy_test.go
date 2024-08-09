package main

import (
	"testing"
	// "strings"
	// "github.com/google/go-cmp/cmp"
	"github.com/GoogleCloudPlatform/cloud-deploy-samples/custom-targets/util/clouddeploy"
	// "github.com/stretchr/testify/assert"
	//"google.golang.org/api/aiplatform/v1"
)

// Tests that addCommonMetadata populates the DeployResult as expected
func TestDepAddCommonMetadata(t *testing.T) {
	newDeployer := &deployer{}
	deployResult := &clouddeploy.DeployResult{}
	if myMap := deployResult.Metadata; myMap != nil {
		t.Errorf("Expected empty field, received: %s", myMap)
	}
	newDeployer.addCommonMetadata(deployResult)
	if _, exists := deployResult.Metadata[clouddeploy.CustomTargetSourceMetadataKey]; !exists {
		t.Errorf("Error: map missing %s key", clouddeploy.CustomTargetSourceMetadataKey)
	}
	if _, exists := deployResult.Metadata[clouddeploy.CustomTargetSourceSHAMetadataKey]; !exists {
		t.Errorf("Error: map missing %s key", clouddeploy.CustomTargetSourceSHAMetadataKey)
	}
}
