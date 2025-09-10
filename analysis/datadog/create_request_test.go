package main

import (
	"testing"

	datadogV2 "github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"github.com/google/go-cmp/cmp"
)

func TestCreateEventsListRequestValid(t *testing.T) {
	testCases := []struct {
		name          string
		query         string
		startTime     string
		endTime       string
		expectedQuery string
	}{
		{
			name:          "simple query",
			query:         "test query",
			startTime:     "2023-01-01T00:00:00Z",
			endTime:       "2023-01-02T01:00:00Z",
			expectedQuery: "test query AND status:error",
		},
		{
			name:          "query with special characters",
			query:         "@monitor_id:123",
			startTime:     "2023-01-01T00:00:00Z",
			endTime:       "2023-01-02T01:00:00Z",
			expectedQuery: "@monitor_id:123 AND status:error",
		},
		{
			name:          "query with existing status:error",
			query:         "status:error",
			startTime:     "2023-01-01T00:00:00Z",
			endTime:       "2023-01-02T01:00:00Z",
			expectedQuery: "status:error AND status:error",
		},
		{
			name:          "query with tags",
			query:         "tags:env:prod,region:us-central1",
			startTime:     "2023-01-01T00:00:00Z",
			endTime:       "2023-01-02T01:00:00Z",
			expectedQuery: "tags:env:prod,region:us-central1 AND status:error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			want := &datadogV2.EventsListRequest{
				Filter: &datadogV2.EventsQueryFilter{
					Query: &tc.expectedQuery,
					From:  &tc.startTime,
					To:    &tc.endTime,
				},
			}
			got, err := createEventsListRequest(tc.query, tc.startTime, tc.endTime)
			if err != nil {
				t.Fatalf("createEventsListRequest() failed: %v", err)
			}

			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("createEventsListRequest() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCreateEventsListRequestInvalid(t *testing.T) {
	testCases := []struct {
		name        string
		query       string
		startTime   string
		endTime     string
		expectedErr string
	}{
		{
			name:        "empty query",
			query:       "",
			startTime:   "2023-01-01T00:00:00Z",
			endTime:     "2023-01-02T01:00:00Z",
			expectedErr: "query is empty",
		},
		{
			name:        "empty start time",
			query:       "@monitor_id:123 AND status:error",
			startTime:   "",
			endTime:     "2023-01-02T01:00:00Z",
			expectedErr: "start time is empty",
		},
		{
			name:        "empty end time",
			query:       "@monitor_id:123",
			startTime:   "2023-01-01T00:00:00Z",
			endTime:     "",
			expectedErr: "end time is empty",
		},
		{
			name:        "start time is after end time",
			query:       "test query",
			startTime:   "2023-01-02T00:00:00Z",
			endTime:     "2023-01-01T00:00:00Z",
			expectedErr: "start time is after end time",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := createEventsListRequest(tc.query, tc.startTime, tc.endTime)
			if err == nil {
				t.Errorf("createEventsListRequest() with %s expected an error, but got none", tc.name)
			}
			if err != nil && err.Error() != tc.expectedErr {
				t.Errorf("createEventsListRequest() with %s returned wrong error: got %q, want %q", tc.name, err.Error(), tc.expectedErr)
			}
		})
	}
}
