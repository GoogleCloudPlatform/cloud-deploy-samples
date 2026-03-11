package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// MonitorDetailsResponse represents some of the fields from the Datadog Get Monitor Details API.
// See https://docs.datadoghq.com/api/latest/monitors/#get-a-monitors-details
type MonitorDetailsResponse struct {
	Name         string `json:"name,omitempty"`
	OverallState string `json:"overall_state,omitempty"`
}

// fetchMonitorDetails fetches details for a specific Datadog monitor
func fetchMonitorDetails(ctx context.Context, monitorID int64, siteURL, apiKey, appKey string) (*MonitorDetailsResponse, error) {
	apiURL, err := SiteToAPIURL(siteURL)
	if err != nil {
		return nil, fmt.Errorf("could not get Datadog API URL: %w", err)
	}
	url := fmt.Sprintf("%s/api/v1/monitor/%d", apiURL, monitorID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("DD-API-KEY", apiKey)
	req.Header.Set("DD-APPLICATION-KEY", appKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-OK status code: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var monitorDetails MonitorDetailsResponse
	if err := json.Unmarshal(bodyBytes, &monitorDetails); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &monitorDetails, nil
}
