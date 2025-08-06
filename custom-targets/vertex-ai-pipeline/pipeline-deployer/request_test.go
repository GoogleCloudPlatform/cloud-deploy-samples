package main

import (
	"context"
	"os"
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/cloud-deploy-samples/custom-targets/util/clouddeploy"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/api/aiplatform/v1"
)

func TestCreateRequestHandlerValidRequest(t *testing.T) {
	aiService, _ := newAIPlatformService(context.Background(), "us-central1")
	storageClient := &storage.Client{}
	testParams := &params{}
	testProject := "test-project"
	testLocation := "us-central1"
	testPipeline := "test-pipeline"
	testTarget := "test-target"
	testPhase := "test-phase"
	testPercentage := 100
	testStorageType := "GCS"
	testInputGCSPath := `gs://test-bucket/test-file.txt`
	testOutputGCSPath := `gs://test-bucket/test-output.txt`
	testWorkloadType := "CB"
	testServiceAccount := "test-service-account"
	testWorkerPool := "test-worker-pool"
	testRelease := "test-release"
	testRollout := "test-rollout"
	testSkaffoldGCSPath := `gs://test-bucket/test-skaffold.yaml`
	testManifestGCSPath := `gs://test-bucket/test-manifest.yaml`

	renderRequest := &clouddeploy.RenderRequest{
		Project:       testProject,
		Location:      testLocation,
		Pipeline:      testPipeline,
		Target:        testTarget,
		Phase:         testPhase,
		Percentage:    testPercentage,
		StorageType:   testStorageType,
		InputGCSPath:  testInputGCSPath,
		OutputGCSPath: testOutputGCSPath,
		WorkloadType:  testWorkloadType,
		WorkloadCBInfo: clouddeploy.CloudBuildWorkload{
			ServiceAccount: testServiceAccount,
			WorkerPool:     testWorkerPool,
		},
	}

	deployRequest := &clouddeploy.DeployRequest{
		Project:         testProject,
		Location:        testLocation,
		Pipeline:        testPipeline,
		Release:         testRelease,
		Rollout:         testRollout,
		Target:          testTarget,
		Phase:           testRollout,
		Percentage:      testPercentage,
		StorageType:     testStorageType,
		InputGCSPath:    testInputGCSPath,
		SkaffoldGCSPath: testSkaffoldGCSPath,
		ManifestGCSPath: testManifestGCSPath,
		OutputGCSPath:   testOutputGCSPath,
		WorkloadType:    testWorkloadType,
		WorkloadCBInfo: clouddeploy.CloudBuildWorkload{
			ServiceAccount: testServiceAccount,
			WorkerPool:     testWorkerPool,
		},
	}

	tests := []struct {
		name               string
		cloudDeployRequest any
		params             *params
		client             *storage.Client
		service            *aiplatform.Service
		wantRequestHandler requestHandler
	}{
		{
			name:               "works with Render Request Handler",
			cloudDeployRequest: renderRequest,
			params:             testParams,
			client:             storageClient,
			service:            aiService,
			wantRequestHandler: &renderer{
				gcsClient:         storageClient,
				aiPlatformService: aiService,
				params:            testParams,
				req:               renderRequest,
			},
		},
		{
			name:               "works with Deploy Request Handler",
			cloudDeployRequest: deployRequest,
			params:             testParams,
			client:             storageClient,
			service:            aiService,
			wantRequestHandler: &deployer{
				gcsClient:         storageClient,
				aiPlatformService: aiService,
				params:            testParams,
				req:               deployRequest,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req, err := createRequestHandler(test.cloudDeployRequest, test.params, test.client, test.service)
			if err != nil {
				t.Errorf("createRequestHandler() returned an error: %v", err)
			}

			opts := []cmp.Option{
				cmp.AllowUnexported(renderer{}, deployer{}, params{}, storage.Client{}, aiplatform.Service{}), // Allow comparing unexported fields
			}

			if diff := cmp.Diff(test.wantRequestHandler, req, opts...); diff != "" {
				t.Errorf("createRequestHandler() returned diff (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCreateRequestHandlerInvalidRequest(t *testing.T) {
	tests := []struct {
		name               string
		cloudDeployRequest any
		wantErrorSubstring string
	}{
		{
			// This function expects a RenderRequest or DeployRequest
			name:               "fails when given random struct",
			cloudDeployRequest: &clouddeploy.RenderResult{},
			wantErrorSubstring: `received unsupported cloud deploy request type: ""`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := createRequestHandler(test.cloudDeployRequest, &params{}, &storage.Client{}, &aiplatform.Service{})
			if err == nil {
				t.Fatalf("createRequestHandler() got err = nil, want %v", test.wantErrorSubstring)
			}

			if !strings.Contains(err.Error(), test.wantErrorSubstring) {
				t.Fatalf("createRequestHandler() returned error (%v) want %v", err, test.wantErrorSubstring)
			}
		})
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
