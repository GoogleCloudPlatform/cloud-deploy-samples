// Package alertwindow provides utility functions for getting the alert time window.
package alertwindow

import (
	"context"
	"fmt"
	"os"
	"time"

	cdenv "github.com/GoogleCloudPlatform/cloud-deploy-samples/packages/cdenv"
	cdapi "google.golang.org/api/clouddeploy/v1"
)

// AlertTimeWindow holds the rollout start time and the end time to look for alerts (which is now).
type AlertTimeWindow struct {
	// StartTime is the start time of the alert window in RFC3339 format.
	StartTime time.Time
	// EndTime is the end time of the alert window in RFC3339 format.
	EndTime time.Time
}

// TimeWindow returns the start time of the rollout as a string representing the number of
// milliseconds since the Unix epoch (this is the time format expected by Datadog).
func TimeWindow(ctx context.Context) (*AlertTimeWindow, error) {
	// Construct the  rollout resource name.
	projectID := os.Getenv(cdenv.ProjectIDEnvKey)
	location := os.Getenv(cdenv.LocationEnvKey)
	pipelineID := os.Getenv(cdenv.PipelineEnvKey)
	releaseID := os.Getenv(cdenv.ReleaseEnvKey)
	rolloutID := os.Getenv(cdenv.RolloutEnvKey)

	rolloutName := fmt.Sprintf("projects/%s/locations/%s/deliveryPipelines/%s/releases/%s/rollouts/%s",
		projectID, location, pipelineID, releaseID, rolloutID)

	cdService, err := cdapi.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to create Cloud Deploy API service: %v", err)
	}

	rollout, err := cdService.Projects.Locations.DeliveryPipelines.Releases.Rollouts.Get(rolloutName).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to get rollout from Cloud Deploy API: %v", err)
	}

	parsedStartTime, err := time.Parse(time.RFC3339, rollout.DeployStartTime)
	if err != nil {
		return nil, fmt.Errorf("unable to convert start time to RFC3339 format in order to validate time")
	}

	endTime := time.Now().Format(time.RFC3339)
	parsedEndTime, err := time.Parse(time.RFC3339, endTime)
	if err != nil {
		return nil, fmt.Errorf("unable to convert start time to RFC3339 format in order to validate time")
	}
	if parsedStartTime.After(parsedEndTime) {
		return nil, fmt.Errorf("start time is after end time")
	}

	return &AlertTimeWindow{StartTime: parsedStartTime, EndTime: parsedEndTime}, nil
}
