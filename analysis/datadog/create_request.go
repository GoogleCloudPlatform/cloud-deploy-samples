package main

import (
	"fmt"
	"time"

	datadogV2 "github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
)

func createEventsListRequest(query, startTime, endTime string) (*datadogV2.EventsListRequest, error) {
	if query == "" {
		return nil, fmt.Errorf("query is empty")
	}
	if startTime == "" {
		return nil, fmt.Errorf("start time is empty")
	}
	if endTime == "" {
		return nil, fmt.Errorf("end time is empty")
	}

	// Parsing the start and end times to validate that the start time is before the end time.
	parsedStartTime, err := time.Parse(time.RFC3339, startTime)
	if err != nil {
		return nil, fmt.Errorf("unable to convert start time to RFC3339 format in order to validate time")
	}
	parsedEndTime, err := time.Parse(time.RFC3339, endTime)
	if err != nil {
		return nil, fmt.Errorf("unable to convert start time to RFC3339 format in order to validate time")
	}
	if parsedStartTime.After(parsedEndTime) {
		return nil, fmt.Errorf("start time is after end time")
	}

	// Appending "status:error" to the query so that only alerts that are firing will be returned

	query += " AND status:error"

	return &datadogV2.EventsListRequest{
		Filter: &datadogV2.EventsQueryFilter{
			Query: &query,
			From:  &startTime,
			To:    &endTime,
		},
	}, nil
}
