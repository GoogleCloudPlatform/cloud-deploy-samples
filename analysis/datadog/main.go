// Package main implements a sample datadog container. It can be used in conjunction with the
// upcoming analysis feature to query datadog for alerts.
package main

import (
	"context"
	"fmt"
	"net/url"
	"os"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/storage"
	datadog "github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/GoogleCloudPlatform/cloud-deploy-samples/analysis/util"
	"github.com/GoogleCloudPlatform/cloud-deploy-samples/packages/secrets"
)

const (
	// customAnalysisKey is the key for the key/value pair added to the metadata to indicate that this sample was used.
	customAnalysisKey = "custom-analysis-type"
	// customAnalysisValue is the value for the key/value pair added to the metadata to indicate that this sample was used.
	customAnalysisValue = "datadog"
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

	// The Datadog client expects a site (e.g. "us5.datadoghq.com"), but the environment
	// variable is the full URL. We parse the URL to extract the host and set it in the
	// context for the API client calls.
	site := evs.SiteURL
	if parsedURL, err := url.Parse(evs.SiteURL); err == nil && parsedURL.Scheme != "" && parsedURL.Host != "" {
		site = parsedURL.Host
	}
	ctx = context.WithValue(
		ctx,
		datadog.ContextServerVariables,
		map[string]string{
			"site": site,
		},
	)

	configuration := datadog.NewConfiguration()
	apiClient := datadog.NewAPIClient(configuration)
	datadogClient := NewDatadogAPIClient(ctx, apiClient)

	// Step 4. Get the alert time window.
	atw, err := analysisutil.TimeWindow(ctx)
	if err != nil {
		return fmt.Errorf("unable to get rollout start time: %v", err)
	}
	// Convert the time into Unix epoch time, which is the format required by Datadog.
	startTimeUnix := fmt.Sprintf("%d", atw.StartTime.UnixMilli())
	endTimeUnix := fmt.Sprintf("%d", atw.EndTime.UnixMilli())

	// Step 5. Query for alerts.
	queryParams := &QueryInfo{
		datadogClient:    datadogClient,
		evs:              evs,
		rolloutStartTime: startTimeUnix,
		endTime:          endTimeUnix,
		apiKey:           apiSecretData,
		appKey:           appSecretData,
	}
	analysisResult, err := queryForAlerts(ctx, queryParams)
	if err != nil {
		// If there was an error querying for alerts, create a new analysis result to upload.
		analysisResult = &analysisutil.AnalysisMetadata{
			Metadata: map[string]string{
				"analysisVendor":  Datadog,
				customAnalysisKey: customAnalysisValue,
				"failureMessage":  err.Error(),
			},
		}
	}

	// Step 6. Upload the result to GCS.
	gcsClient, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("unable to create GCS client: %v", err)
	}
	if err := analysisutil.UploadResult(ctx, analysisResult, gcsClient); err != nil {
		return fmt.Errorf("unable to upload result to GCS: %v", err)
	}

	// Return an error so the build fails if there are any alerts firing.
	if analysisResult.Metadata["failureMessage"] != "" {
		return fmt.Errorf("%s", analysisResult.Metadata["failureMessage"])
	}
	return nil
}
