// Copyright 2023 Google LLC

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     https://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	config "cloud.google.com/go/config/apiv1"
	"cloud.google.com/go/config/apiv1/configpb"
	retry "github.com/avast/retry-go/v4"
)

// getDeployment gets the Deployment.
func getDeployment(ctx context.Context, client *config.Client, deploymentName string) (*configpb.Deployment, error) {
	req := &configpb.GetDeploymentRequest{
		Name: deploymentName,
	}
	return client.GetDeployment(ctx, req)
}

// pollDeploymentUntilTerminal repeatedly calls GetDeployment until all retry attempts are consumed or the Deployment
// reaches a terminal state. If the latest revision provided changes on the Deployment while polling then an error
// is returned.
func pollDeploymentUntilTerminal(ctx context.Context, client *config.Client, deploymentName string, latestRevision string) (*configpb.Deployment, error) {
	attempts := 0
	dep, err := retry.DoWithData(
		func() (*configpb.Deployment, error) {
			attempts++
			dep, err := getDeployment(ctx, client, deploymentName)
			if err != nil {
				return nil, err
			}
			if dep.LatestRevision != latestRevision {
				return nil, fmt.Errorf("latest revision changed from %s to %s", latestRevision, dep.LatestRevision)
			}
			state := dep.State
			fmt.Printf("Deployment %s state is %s\n", deploymentName, state.String())
			if isSucceededDeployment(state) || isFailedDeployment(state) {
				return dep, nil
			} else if isInProgressDeployment(state) {
				return nil, errors.New("deployment still in progress")
			}
			return nil, fmt.Errorf("unknown deployment state %s", state)
		},
		// Keep retrying only if Deployment was retrieved and is still in progress.
		retry.RetryIf(func(err error) bool {
			return err.Error() == "deployment still in progress"
		}),
		retry.Attempts(20),
		retry.Delay(30*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("error polling deployment until terminal state after %d attempts: %v", attempts, err)
	}
	return dep, nil
}

// createDeployment creates the Deployment and waits for the LRO to complete. While waiting for the LRO
// to complete the Deployment is periodically retrieved in order to log a state update.
func createDeployment(ctx context.Context, client *config.Client, deployment *configpb.Deployment) (*configpb.Deployment, error) {
	// Name is "projects/{project}/locations/{location}/deployments/{deployment}".
	nameParts := strings.Split(deployment.Name, "/")
	op, err := client.CreateDeployment(ctx, &configpb.CreateDeploymentRequest{
		Parent:       fmt.Sprintf("projects/%s/locations/%s", nameParts[1], nameParts[3]),
		DeploymentId: nameParts[5],
		Deployment:   deployment,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating infrastructure manager deployment: %v", err)
	}
	fmt.Printf("Waiting on create Deployment operation %s\n", op.Name())
	var d *configpb.Deployment
	for {
		time.Sleep(30 * time.Second)
		pd, err := op.Poll(ctx)
		if err != nil {
			return nil, fmt.Errorf("error polling create deployment operation: %v", err)
		}
		if pd != nil {
			d = pd
			break
		}
		// If the operation isn't complete then get the Deployment to log the current state.
		tempD, err := getDeployment(ctx, client, deployment.Name)
		if err != nil {
			return nil, fmt.Errorf("error getting deployment: %v", err)
		}
		fmt.Printf("Create operation still in progress, current Deployment state: %s\n", tempD.State)
	}
	return d, nil
}

// updateDeployment updates the Deployment and waits for the LRO to complete. While waiting for the LRO
// to complete the Deployment is periodically retrieved in order to log a state update.
func updateDeployment(ctx context.Context, client *config.Client, renderedDeployment *configpb.Deployment) (*configpb.Deployment, error) {
	op, err := client.UpdateDeployment(ctx, &configpb.UpdateDeploymentRequest{
		Deployment: renderedDeployment,
	})
	if err != nil {
		return nil, fmt.Errorf("error calling update deployment: %v", err)
	}
	fmt.Printf("Waiting on update Deployment operation %s\n", op.Name())
	var d *configpb.Deployment
	for {
		time.Sleep(30 * time.Second)
		pd, err := op.Poll(ctx)
		if err != nil {
			return nil, fmt.Errorf("error polling create deployment operation: %v", err)
		}
		if pd != nil {
			d = pd
			break
		}
		// If the operation isn't complete then get the Deployment to log the current state.
		tempD, err := getDeployment(ctx, client, renderedDeployment.Name)
		if err != nil {
			return nil, fmt.Errorf("error getting deployment: %v", err)
		}
		fmt.Printf("Update operation still in progress, current Deployment state: %s", tempD.State)
	}
	return d, nil
}

// isInProgressDeployment returns whether the Deployment state is considered to be in progress by the deployer.
func isInProgressDeployment(state configpb.Deployment_State) bool {
	return state == configpb.Deployment_CREATING || state == configpb.Deployment_UPDATING
}

// isSucceededDeployment returns whether the Deployment state is considered to be succeeded by the deployer.
func isSucceededDeployment(state configpb.Deployment_State) bool {
	return state == configpb.Deployment_ACTIVE
}

// isFailedDeployment returns whether the Deployment state is considered to be failed by the deployer.
func isFailedDeployment(state configpb.Deployment_State) bool {
	switch state {
	case configpb.Deployment_FAILED,
		configpb.Deployment_SUSPENDED,
		configpb.Deployment_DELETED,
		configpb.Deployment_DELETING:
		return true
	}
	return false
}

// getRevision gets the Revision.
func getRevision(ctx context.Context, client *config.Client, revisionName string) (*configpb.Revision, error) {
	req := &configpb.GetRevisionRequest{
		Name: revisionName,
	}
	return client.GetRevision(ctx, req)
}
