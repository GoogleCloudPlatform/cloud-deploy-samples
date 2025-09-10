package main

import (
	"fmt"
	"time"
)

// queryForAlerts queries datadog for alerts. It returns an AnalysisResult or an error if one is
// encountered during the process. The returned AnalysisResult will have a "FAILED" status if an
// alert is found.
func queryForAlerts(datadogClient *DatadogAPIClient, evs *ValidatedEnvVars, rolloutStartTime string) (*AnalysisResult, error) {
	var parsedResponse *AnalysisResult
	for _, query := range evs.Queries {
		endTime := time.Now().Format(time.RFC3339)
		request, err := createEventsListRequest(query, rolloutStartTime, endTime)
		if err != nil {
			return nil, fmt.Errorf("unable to create events list request: %w", err)
		}

		response, err := datadogClient.SearchEvents(request)
		if err != nil {
			return nil, fmt.Errorf("unable to search events: %w", err)
		}

		parsedResponse, err = parseDatadogResponse(response, evs.Location, query)
		if err != nil {
			return nil, fmt.Errorf("unable to parse datadog response: %w", err)
		}

		if parsedResponse.ResultStatus == "FAILED" {
			// An alert was found, this is the final result.
			return parsedResponse, nil
		}
	}
	// If we get here, all queries succeeded without finding alerts.
	// Return the result of the last query.
	return parsedResponse, nil
}
