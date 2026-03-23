package main

import (
	"fmt"

	datadogV2 "github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
)

// FailureInfo represents the failure message and the URL of the monitor that is alerting. This is
// used to populate the AnalysisMetadata.
type FailureInfo struct {
	FailureMessage string `json:"failureMessage,omitempty"`
	MonitorID      int64  `json:"monitorID,omitempty"`
	URL            string `json:"url,omitempty"`
}

// parseDatadogResponse parses the Datadog response and returns a FailureInfo struct populated with
// the failure message and the URL of the monitor that is alerting. If there are no alerts firing,
// an empty FailureInfo struct is returned.
func parseDatadogResponse(response *datadogV2.EventsListResponse, query string, siteURL string) *FailureInfo {
	// If there is no data in the response, there are no alerts firing, so this is a success.
	if len(response.Data) == 0 {
		return &FailureInfo{}
	}

	// Since the query filters for "status:error", any event in the response is a failure.
	// We use the first event to populate the result.
	firstEvent := response.GetData()[0]
	attributes := firstEvent.GetAttributes()
	nestedAttributes := attributes.GetAttributes()
	monitor := nestedAttributes.GetMonitor()
	monitorID := nestedAttributes.GetMonitorId()
	url := siteURL + "/monitors/" + fmt.Sprint(monitorID)
	return &FailureInfo{
		FailureMessage: monitor.GetMessage(),
		MonitorID:      monitorID,
		URL:            url,
	}
}
