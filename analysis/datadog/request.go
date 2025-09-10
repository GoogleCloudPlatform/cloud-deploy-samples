package main

import (
	"context"

	datadogV2 "github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	datadog "google3/third_party/golang/github_com/DataDog/datadog_api_client_go/v/v2/api/datadog/datadog"
)

// DatadogClient is an interface for interacting with the Datadog API and allows for mocking in tests.
type DatadogClient interface {
	SearchEvents(ctx context.Context, req *datadogV2.EventsListRequest) (*datadogV2.EventsListResponse, error)
}

// DatadogAPIClient implements the DatadogClient interface.
type DatadogAPIClient struct {
	client *datadog.APIClient
	ctx    context.Context
}

// NewDatadogAPIClient creates a new DatadogAPIClient.
func NewDatadogAPIClient(ctx context.Context, apiClient *datadog.APIClient) *DatadogAPIClient {
	return &DatadogAPIClient{
		client: apiClient,
		ctx:    ctx,
	}
}

// SearchEvents calls the Datadog Search Events API.
func (c *DatadogAPIClient) SearchEvents(req *datadogV2.EventsListRequest) (*datadogV2.EventsListResponse, error) {
	api := datadogV2.NewEventsApi(c.client)
	resp, _, err := api.SearchEvents(c.ctx, *datadogV2.NewSearchEventsOptionalParameters().WithBody(*req))
	if err != nil {
		return nil, err
	}
	return &resp, nil
}
