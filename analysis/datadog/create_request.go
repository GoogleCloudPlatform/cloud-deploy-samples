package main

import (
	"fmt"

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
