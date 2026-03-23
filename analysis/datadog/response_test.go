package main

import (
	"testing"

	datadog "github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	datadogV2 "github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
)

func TestParseDatadogResponse(t *testing.T) {
	testMessage := "test-message"
	testQuery := "test-query"
	testSite := "test-site"
	testMonitorIDVal := int64(12345)
	testMonitorID := *datadog.NewNullableInt64(&testMonitorIDVal)
	tests := []struct {
		name               string
		response           *datadogV2.EventsListResponse
		wantFailureMessage string
		wantURL            string
		wantErr            bool
	}{
		{
			name: "succeeded",
			response: &datadogV2.EventsListResponse{
				Data: []datadogV2.EventResponse{},
			},
			wantFailureMessage: "",
			wantURL:            "",
		},
		{
			name: "failed",
			response: &datadogV2.EventsListResponse{
				Data: []datadogV2.EventResponse{
					{
						Attributes: &datadogV2.EventResponseAttributes{
							Attributes: &datadogV2.EventAttributes{
								Monitor: *datadogV2.NewNullableMonitorType(&datadogV2.MonitorType{
									Message: &testMessage,
								}),
								MonitorId: testMonitorID,
							},
						},
					},
				},
			},
			wantFailureMessage: "test-message",
			wantURL:            "test-site/monitors/12345",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseDatadogResponse(tc.response, testQuery, testSite)
			if tc.wantFailureMessage != got.FailureMessage {
				t.Errorf("parseDatadogResponse() failure message mismatch - got: %s, want: %s", got.FailureMessage, tc.wantFailureMessage)
			}
			if tc.wantURL != got.URL {
				t.Errorf("parseDatadogResponse() URL mismatch - got: %s, want: %s", got.URL, tc.wantURL)
			}
		})
	}
}
