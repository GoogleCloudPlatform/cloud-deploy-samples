package main

import (
	"context"
	"fmt"
	"os"
	"time"

	cdenv "github.com/GoogleCloudPlatform/cloud-deploy-samples/packages/cdenv"
	cdapi "google.golang.org/api/clouddeploy/v1"
)

// convertTime converts an RFC3339 formatted time string to a string representing
// the number of milliseconds since the Unix epoch.
func convertTime(t string) (string, error) {
	if t == "" {
		return "", fmt.Errorf("time string is empty")
	}
	parsedTime, err := time.Parse(time.RFC3339, t)
	if err != nil {
		return "", fmt.Errorf("time.Parse: %w", err)
	}
	return fmt.Sprintf("%d", parsedTime.UnixMilli()), nil
}

// rolloutStartTime returns the start time of the rollout as a string representing the number of
// milliseconds since the Unix epoch.
func rolloutStartTime(ctx context.Context) (string, error) {
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
		return "", fmt.Errorf("unable to create Cloud Deploy API service: %v", err)
	}

	rollout, err := cdService.Projects.Locations.DeliveryPipelines.Releases.Rollouts.Get(rolloutName).Do()
	if err != nil {
		return "", fmt.Errorf("unable to get rollout from Cloud Deploy API: %v", err)
	}

	timeString := rollout.DeployStartTime
	// Convert the time into Unix epoch time, which is the format required by Datadog.
	convertedTime, err := convertTime(timeString)
	if err != nil {
		return "", fmt.Errorf("could not convert rollout deploy start time '%s': %w", timeString, err)
	}

	return convertedTime, nil
}
