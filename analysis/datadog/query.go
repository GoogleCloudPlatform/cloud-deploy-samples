package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/GoogleCloudPlatform/cloud-deploy-samples/analysis/util"
)

const (
	// Datadog is the analysis third party provider.
	Datadog = "Datadog"
)

// formatTimestamp converts a unix timestamp in milliseconds to a human-readable
// format. If the timestamp cannot be parsed, it is returned as is.
func formatTimestamp(timestampStr string) string {
	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return timestampStr
	}
	return time.UnixMilli(timestamp).Format(time.RFC1123)
}

// QueryInfo contains the parameters to query for Datadog alerts to make a Datadog SearchEvents
// API call.
type QueryInfo struct {
	datadogClient    *DatadogAPIClient
	evs              *ValidatedEnvVars
	rolloutStartTime string
	endTime          string
	apiKey           string
	appKey           string
}

// queryForAlerts queries datadog for alerts. It returns an AnalysisMetadata or an error if one is
// encountered during the process. The returned AnalysisMetadata will have a failureMessage if an
// alert is found.
func queryForAlerts(ctx context.Context, params *QueryInfo) (*analysisutil.AnalysisMetadata, error) {
	analysisResult := &analysisutil.AnalysisMetadata{
		Metadata: map[string]string{
			"analysisVendor":  Datadog,
			customAnalysisKey: customAnalysisValue,
		},
	}

	for _, query := range params.evs.Queries {
		request, err := createEventsListRequest(query, params.rolloutStartTime, params.endTime)
		if err != nil {
			return nil, fmt.Errorf("unable to create events list request: %w", err)
		}
		response, err := params.datadogClient.SearchEvents(request)
		if err != nil {
			return nil, fmt.Errorf("unable to search events: %w", err)
		}
		analysisResult.Metadata["query"] = query
		parsedResponse := parseDatadogResponse(response, query, params.evs.SiteURL)
		// If FailureMessage is empty, it means no alert was found for this query. Continue to the next query.
		if parsedResponse.FailureMessage == "" {
			continue
		}
		// Check to see if the alert is still firing.
		currentStatus, err := fetchMonitorDetails(ctx, parsedResponse.MonitorID, params.evs.SiteURL, params.apiKey, params.appKey)
		if err != nil {
			return nil, fmt.Errorf("unable to fetch monitor details for monitor ID %d: %w", parsedResponse.MonitorID, err)
		}
		// The monitor is still currently alerting, so this is the final result.
		if currentStatus.OverallState == "Alert" {
			fmt.Printf("Datadog alerts were found for the following query: %q. Queried from %s to %s\n", query, formatTimestamp(*request.Filter.From), formatTimestamp(*request.Filter.To))
			fmt.Printf("The name of the monitor that is alerting is: %s\n", currentStatus.Name)
			fmt.Printf("The url of the monitor that is alerting is: %s\n", parsedResponse.URL)
			analysisResult.Metadata["failureMessage"] = parsedResponse.FailureMessage
			analysisResult.Metadata["url"] = parsedResponse.URL
			return analysisResult, nil
		}
	}

	fmt.Printf("No Datadog alerts found from %s to %s\n", formatTimestamp(params.rolloutStartTime), formatTimestamp(params.endTime))
	// If we get here, all queries succeeded without finding alerts.
	return analysisResult, nil
}
