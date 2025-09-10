// Package main implements a sample datadog container. It can be used in conjunction with the
// upcoming analysis feature to query datadog for alerts.
// IMPORTANT NOTE: This is a work in progress and not ready for production use.
package main

import (
	"context"
	"fmt"
	"os"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/cloud-deploy-samples/packages/secrets"
	datadog "google3/third_party/golang/github_com/DataDog/datadog_api_client_go/v/v2/api/datadog/datadog"
)

func main() {
	if err := do(); err != nil {
		fmt.Printf("err: %v\n", err)
		os.Exit(1)
	}
}

func do() error {
	ctx := context.Background()

	// Step 1. Validate environment variables.
	evs, err := envVars()
	if err != nil {
		return err
	}

	// Step 2. Get the secret using the Secret Manager API and the env var they provided.
	smClient, err := secretmanager.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("unable to create secret manager client: %v", err)
	}
	apiSecretData, err := secrets.SecretVersionData(ctx, evs.APISecret, smClient)
	if err != nil {
		return fmt.Errorf("unable to access datadog API secret: %v", err)
	}
	appSecretData, err := secrets.SecretVersionData(ctx, evs.AppSecret, smClient)
	if err != nil {
		return fmt.Errorf("unable to access datadog app secret: %v", err)
	}

	// Step 3. Create the datadog client.
	ctx = context.WithValue(
		ctx,
		datadog.ContextAPIKeys,
		map[string]datadog.APIKey{
			"apiKeyAuth": {
				Key: apiSecretData,
			},
			"appKeyAuth": {
				Key: appSecretData,
			},
		},
	)

	configuration := datadog.NewConfiguration()
	apiClient := datadog.NewAPIClient(configuration)
	datadogClient := NewDatadogAPIClient(ctx, apiClient)

	// Step 4. Get the rollout start time.
	rolloutStartTime, err := rolloutStartTime(ctx)
	if err != nil {
		return fmt.Errorf("unable to get rollout start time: %v", err)
	}
	fmt.Printf("rollout start time: %s\n", rolloutStartTime)

	// Step 5. Query for alerts.
	analysisResult, err := queryForAlerts(datadogClient, evs, rolloutStartTime)
	if err != nil {
		analysisResult = &AnalysisResult{
			ResultStatus:   "FAILED",
			FailureMessage: err.Error(),
			AnalysisVendor: "Datadog",
		}
	}

	// Step 6. Upload the result to GCS.
	gcsClient, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("unable to create GCS client: %v", err)
	}
	if err := uploadResult(ctx, analysisResult, gcsClient); err != nil {
		return fmt.Errorf("unable to upload result to GCS: %v", err)
	}

	// Returning an error so the build fails if there are any alerts firing
	if analysisResult.ResultStatus == "FAILED" {
		return fmt.Errorf("%s", analysisResult.FailureMessage)
	}
	return nil
}
