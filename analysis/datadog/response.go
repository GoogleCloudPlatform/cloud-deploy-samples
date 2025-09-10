package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"cloud.google.com/go/storage"
	datadogV2 "github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	cdenv "github.com/GoogleCloudPlatform/cloud-deploy-samples/packages/cdenv"
	"github.com/GoogleCloudPlatform/cloud-deploy-samples/packages/gcs"
)

// Metadata contains metadata associated with the analysis.
type Metadata struct {
	// Query is the Datadog query that was executed to determine if any alerts were firing.
	Query string `json:"query,omitempty"`
	// TODO(b/443960479): Uncomment this field once we are able to get the path from Datadog API.
	// AlertURL is the Datadog URL to use to view the alert.
	// AlertURL string `json:"alertURL,omitempty"`
}

// AnalysisResult represents the response that will be uploaded to GCS.
type AnalysisResult struct {
	// ResultStatus is the status of the analysis result. Valid values are "SUCCEEDED" or "FAILED".
	ResultStatus string `json:"resultStatus"`
	// AnalysisVendor is the name of the 3rd party system being queried.
	AnalysisVendor string `json:"analysisVendor,omitempty"`
	// FailureMessage is the failure message.
	FailureMessage string `json:"failureMessage,omitempty"`
	// Metadata contains metadata associated with the analysis result.
	Metadata *Metadata `json:"metadata,omitempty"`
}

func parseDatadogResponse(response *datadogV2.EventsListResponse, location string, query string) (*AnalysisResult, error) {
	// If there is no data in the response, there are no alerts firing, so this is a success.
	if len(response.Data) == 0 {
		return &AnalysisResult{
			ResultStatus:   "SUCCEEDED",
			AnalysisVendor: "Datadog",
		}, nil
	}

	// Since the query filters for "status:error", any event in the response is a failure.
	// We use the first event to populate the result.
	firstEvent := response.GetData()[0]
	attributes := firstEvent.GetAttributes()
	nestedAttributes := attributes.GetAttributes()
	monitor := nestedAttributes.GetMonitor()
	message := monitor.GetMessage()

	// TODO(b/443960479): Uncomment this code once we are able to get the path from Datadog API.
	// If a location is not provided, the alert URL is empty.
	// If a location is provided, prepend the base URL to the path from Datadog response.
	// var alertURL string
	// if location != "" {
	// 	baseURL, err := ToSiteURL(location)
	// 	if err != nil {
	// 		return nil, fmt.Errorf("failed to get Datadog base URL: %v", err)
	// 	}

	// 	path := ""
	// 	alertURL = baseURL + path
	// }

	return &AnalysisResult{
		ResultStatus:   "FAILED",
		AnalysisVendor: "Datadog",
		FailureMessage: message,
		Metadata: &Metadata{
			Query: query,
			// TODO(b/443960479): Uncomment this field once we are able to get the path from Datadog API.
			// AlertURL: alertURL,
		},
	}, nil
}

// uploadResult uploads the result to GCS.
func uploadResult(ctx context.Context, result *AnalysisResult, client *storage.Client) error {
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %v", err)
	}
	// Get the GCS URI where the results file should be uploaded.
	uri := os.Getenv(cdenv.OutputGCSEnvKey)
	return gcs.Upload(ctx, client, uri, &gcs.UploadContent{Data: data})
}
