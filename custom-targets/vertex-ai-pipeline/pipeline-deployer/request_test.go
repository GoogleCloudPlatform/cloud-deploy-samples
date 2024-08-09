package main

import (
	"context"
	"fmt"
	"os"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/cloud-deploy-samples/custom-targets/util/clouddeploy"
)

func TestCreateRequestHandler(t *testing.T) {
	aiService, _ := newAIPlatformService(context.Background(), "us-central1")
	req, err := createRequestHandler(&clouddeploy.RenderRequest{}, &params{}, &storage.Client{}, aiService)
	if err != nil {
		t.Errorf("Expected: success, Actual: %s", err)
	}
	switch req.(type) {
	case *renderer:
		fmt.Println("Handler is a renderer")
	default:
		t.Errorf("Expected: renderer, Actual: uknown type")
	}

	req, err = createRequestHandler(&clouddeploy.DeployRequest{}, &params{}, &storage.Client{}, aiService)
	if err != nil {
		t.Errorf("Expected: success, Actual: %s", err)
	}
	switch req.(type) {
	case *deployer:
		fmt.Println("Handler is a deployer")
	default:
		t.Errorf("Expected: deployer, Actual: uknown type")
	}

	req, err = createRequestHandler(&clouddeploy.RenderResult{}, &params{}, &storage.Client{}, aiService)
	if err == nil {
		t.Errorf("Expected: ERROR, Actual: %s", err)
	}
}

func TestDetermineParams(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// Set environment variables
		os.Setenv(pipelineEnvKey, "my-pipeline-name")
		os.Setenv(configPathKey, "folder/file.yaml")
		os.Setenv(paramValsKey, `{"param1": "value1", "param2": "value2"}`)
		os.Setenv(locValsKey, "us-central1")
		os.Setenv(projectValsKey, "my-project-id")

		// Call determineParams
		params, err := determineParams()

		// Assert no error
		if err != nil {
			t.Errorf("determineParams() returned an error: %v", err)
		}

		// Assert expected values
		if params.location != "us-central1" {
			t.Errorf("Expected location to be 'us-central1', got: %s", params.location)
		}
		if params.project != "my-project-id" {
			t.Errorf("Expected project to be 'my-project-id', got: %s", params.project)
		}
		if params.pipeline != "my-pipeline-name" {
			t.Errorf("Expected pipeline to be 'my-pipeline-name', got: %s", params.pipeline)
		}
		if params.pipelineParams["param1"] != "value1" {
			t.Errorf("Expected pipelineParams['param1'] to be 'value1', got: %s", params.pipelineParams["param1"])
		}
		if params.pipelineParams["param2"] != "value2" {
			t.Errorf("Expected pipelineParams['param2'] to be 'value2', got: %s", params.pipelineParams["param2"])
		}
		if params.configPath != "folder/file.yaml" {
			t.Errorf("Expected environment to be 'folder/file.yaml', got: %s", params.configPath)
		}
	})

	t.Run("EmptyConfigPath", func(t *testing.T) {
		// Set empty environment environment variable
		os.Setenv(configPathKey, "")

		// Call determineParams
		_, err := determineParams()

		// Assert error
		if err == nil {
			t.Errorf("determineParams() should have returned an error, but it didn't")
		}
	})

	t.Run("MissingConfigPath", func(t *testing.T) {
		// Remove environment environment variable
		os.Unsetenv(configPathKey)

		// Call determineParams
		_, err := determineParams()

		// Assert error
		if err == nil {
			t.Errorf("determineParams() should have returned an error, but it didn't")
		}
		os.Setenv(configPathKey, "folder/file.yaml")
	})

	t.Run("EmptyParams", func(t *testing.T) {
		// Set empty params environment variable
		os.Setenv(paramValsKey, "{}")

		// Call determineParams
		_, err := determineParams()

		// Assert error
		if err == nil {
			t.Errorf("determineParams() should have returned an error, but it didn't")
		}
	})

	t.Run("WrongParams", func(t *testing.T) {
		// Remove params environment variable
		os.Setenv(paramValsKey, "{CAN'T UNMARSHALL")

		// Call determineParams
		_, err := determineParams()

		// Assert error
		if err == nil {
			t.Errorf("determineParams() should have returned an error, but it didn't")
		}
	})

	t.Run("MissingParams", func(t *testing.T) {
		// Remove params environment variable
		os.Unsetenv(paramValsKey)

		// Call determineParams
		_, err := determineParams()

		// Assert error
		if err == nil {
			t.Errorf("determineParams() should have returned an error, but it didn't")
		}
		os.Setenv(paramValsKey, `{"param1": "value1", "param2": "value2"}`)
	})

	t.Run("EmptyPipeline", func(t *testing.T) {
		// Set empty pipeline environment variable
		os.Setenv(pipelineEnvKey, "")

		// Call determineParams
		_, err := determineParams()

		// Assert error
		if err == nil {
			t.Errorf("determineParams() should have returned an error, but it didn't")
		}
	})

	t.Run("MissingPipeline", func(t *testing.T) {
		// Remove pipeline environment variable
		os.Unsetenv(pipelineEnvKey)

		// Call determineParams
		_, err := determineParams()

		// Assert error
		if err == nil {
			t.Errorf("determineParams() should have returned an error, but it didn't")
		}
		os.Setenv(pipelineEnvKey, "my-pipeline-name")
	})

	t.Run("EmptyProject", func(t *testing.T) {
		// Set empty project environment variable
		os.Setenv(projectValsKey, "")

		// Call determineParams
		_, err := determineParams()

		// Assert error
		if err == nil {
			t.Errorf("determineParams() should have returned an error, but it didn't")
		}
	})

	t.Run("MissingProject", func(t *testing.T) {
		// Remove project environment variable
		os.Unsetenv(projectValsKey)

		// Call determineParams
		_, err := determineParams()

		// Assert error
		if err == nil {
			t.Errorf("determineParams() should have returned an error, but it didn't")
		}
		os.Setenv(projectValsKey, "my-project-id")
	})

	t.Run("EmptyLocation", func(t *testing.T) {
		// Set empty location environment variable
		os.Setenv(locValsKey, "")

		// Call determineParams
		_, err := determineParams()

		// Assert error
		if err == nil {
			t.Errorf("determineParams() should have returned an error, but it didn't")
		}
	})

	t.Run("MissingLocation", func(t *testing.T) {
		// Remove location environment variable
		os.Unsetenv(locValsKey)

		// Call determineParams
		_, err := determineParams()

		// Assert error
		if err == nil {
			t.Errorf("determineParams() should have returned an error, but it didn't")
		}
		os.Setenv(locValsKey, "us-central1")
	})
}
