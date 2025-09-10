package main

import (
	"testing"

	datadogV2 "github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"github.com/google/go-cmp/cmp"
)

// TODO(b/443960479): Uncomment the 'AlertURL' fields and tests for
// invalid location once we are able to get the path from Datadog API.
func TestParseDatadogResponse(t *testing.T) {
	testMessage := "test-message"
	tests := []struct {
		name     string
		response *datadogV2.EventsListResponse
		location string
		query    string
		want     *AnalysisResult
		wantErr  bool
	}{
		{
			name: "succeeded",
			response: &datadogV2.EventsListResponse{
				Data: []datadogV2.EventResponse{},
			},
			want: &AnalysisResult{
				ResultStatus:   "SUCCEEDED",
				AnalysisVendor: "Datadog",
			},
		},
		{
			name: "failed with location",
			response: &datadogV2.EventsListResponse{
				Data: []datadogV2.EventResponse{
					{
						Attributes: &datadogV2.EventResponseAttributes{
							Attributes: &datadogV2.EventAttributes{
								Monitor: *datadogV2.NewNullableMonitorType(&datadogV2.MonitorType{
									Message: &testMessage,
								}),
							},
						},
					},
				},
			},
			location: "us5",
			query:    "test-query",
			want: &AnalysisResult{
				ResultStatus:   "FAILED",
				AnalysisVendor: "Datadog",
				FailureMessage: "test-message",
				Metadata: &Metadata{
					Query: "test-query",
					// AlertURL: "https://api.us5.datadoghq.com",
				},
			},
		},
		{
			name: "failed without location",
			response: &datadogV2.EventsListResponse{
				Data: []datadogV2.EventResponse{
					{
						Attributes: &datadogV2.EventResponseAttributes{
							Attributes: &datadogV2.EventAttributes{
								Monitor: *datadogV2.NewNullableMonitorType(&datadogV2.MonitorType{
									Message: &testMessage,
								}),
							},
						},
					},
				},
			},
			query: "test-query",
			want: &AnalysisResult{
				ResultStatus:   "FAILED",
				AnalysisVendor: "Datadog",
				FailureMessage: "test-message",
				Metadata: &Metadata{
					Query: "test-query",
					// AlertURL: "",
				},
			},
		},
		// {
		// 	name: "failed with invalid location",
		// 	response: &datadogV2.EventsListResponse{
		// 		Data: []datadogV2.EventResponse{
		// 			{
		// 				Attributes: &datadogV2.EventResponseAttributes{
		// 					Attributes: &datadogV2.EventAttributes{
		// 						Monitor: *datadogV2.NewNullableMonitorType(&datadogV2.MonitorType{
		// 							Message: &testMessage,
		// 						}),
		// 					},
		// 				},
		// 			},
		// 		},
		// 	},
		// 	location: "invalid-location",
		// 	query:    "test-query",
		// 	wantErr:  true,
		// },
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseDatadogResponse(tc.response, tc.location, tc.query)
			if (err != nil) != tc.wantErr {
				t.Errorf("parseDatadogResponse() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("parseDatadogResponse() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
