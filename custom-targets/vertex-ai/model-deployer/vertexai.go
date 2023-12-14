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
	"fmt"
	"google.golang.org/api/aiplatform/v1"
	"google.golang.org/api/option"
	"os"
	"sigs.k8s.io/yaml"
	"strings"
)

// deployModelFromManifest loads the file provided in `path` and returns the parsed DeployModelRequest
// from the data.
func deployModelFromManifest(path string) (*aiplatform.GoogleCloudAiplatformV1DeployModelRequest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading manifest file: %v", err)
	}

	deployModelRequest := &aiplatform.GoogleCloudAiplatformV1DeployModelRequest{}

	if err = yaml.Unmarshal(data, deployModelRequest); err != nil {
		return nil, fmt.Errorf("unable to parse deploy model deployModelRequest from manifest file: %v", err)
	}

	return deployModelRequest, nil
}

// fetchPreviousModel queries the provided Vertex AI endpoint to determine the model that was previously
// deployed.
func fetchPreviousModel(service *aiplatform.Service, endpointName, currentModel string) (string, error) {
	endpoint, err := service.Projects.Locations.Endpoints.Get(endpointName).Do()
	if err != nil {
		return "", fmt.Errorf("unable to fetch endpoint: %v", err)
	}

	deployedModels := map[string]*aiplatform.GoogleCloudAiplatformV1DeployedModel{}

	for _, dm := range endpoint.DeployedModels {
		modelNameWithVersion := resolveDeployedModelNameWithVersion(dm)
		deployedModels[modelNameWithVersion] = dm
	}

	delete(deployedModels, currentModel)

	if len(deployedModels) != 1 {
		return "", fmt.Errorf("unable to resolve previous deployed currentModel to canary against. Not including the current currentModel to be deployed, the endpoint has %d deployed models but expected only one", len(deployedModels))
	}

	var firstModel []*aiplatform.GoogleCloudAiplatformV1DeployedModel

	for _, dm := range deployedModels {
		firstModel = append(firstModel, dm)
	}
	return firstModel[0].Id, nil
}

// resolveDeployedModelNameWithVersion returns the model resource name associated with the  provided DeployedModel
// with its version ID attached.
func resolveDeployedModelNameWithVersion(deployedModel *aiplatform.GoogleCloudAiplatformV1DeployedModel) string {
	if strings.Contains(deployedModel.Model, "@") {
		return deployedModel.Model
	}
	return fmt.Sprintf("%s@%s", deployedModel.Model, deployedModel.ModelVersionId)
}

// resolveModelWithVersion returns the model resource name its version ID attached.
func resolveModelWithVersion(model *aiplatform.GoogleCloudAiplatformV1Model) string {
	if strings.Contains(model.Name, "@") {
		return model.Name
	}
	return fmt.Sprintf("%s@%s", model.Name, model.VersionId)
}

// regionFromModel extracts the region from the model region name.
func regionFromModel(modelName string) (string, error) {
	matches := modelRegex.FindStringSubmatch(modelName)
	if len(matches) == 0 {
		return "", fmt.Errorf("unable to parse model name")
	}

	return matches[2], nil
}

// extracts the region from the endpoint resource name.
func regionFromEndpoint(endpointName string) (string, error) {
	matches := endpointRegex.FindStringSubmatch(endpointName)
	if len(matches) == 0 {
		return "", fmt.Errorf("unable to parse endpoint name")
	}

	return matches[2], nil
}

// newAIPlatformService generates a Service that can make API calls in the specified region.
func newAIPlatformService(ctx context.Context, region string) (*aiplatform.Service, error) {
	endPointOption := option.WithEndpoint(fmt.Sprintf("%s-aiplatform.googleapis.com", region))
	regionalService, err := aiplatform.NewService(ctx, endPointOption)
	if err != nil {
		return nil, fmt.Errorf("unable to authenticate")
	}

	return regionalService, nil
}

// fetchModel calls the aiplatform API to fetch the Vertex AI model using the given model name.
func fetchModel(service *aiplatform.Service, modelName string) (*aiplatform.GoogleCloudAiplatformV1Model, error) {
	model, err := service.Projects.Locations.Models.Get(modelName).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %v", err)
	}
	return model, nil
}

// minReplicaCountFromConfig returns the minReplicaCount value from the provided configuration file.
func minReplicaCountFromConfig(deployedModel *aiplatform.GoogleCloudAiplatformV1DeployedModel) int64 {
	if deployedModel.DedicatedResources != nil {
		return deployedModel.DedicatedResources.MinReplicaCount
	}
	return 0
}

// deployModel performs the DeployModel request and awaits the resulting operation until it completes, it times out or an error occurs.
func deployModel(ctx context.Context, aiPlatformService *aiplatform.Service, endpoint string, request *aiplatform.GoogleCloudAiplatformV1DeployModelRequest) error {
	op, err := aiPlatformService.Projects.Locations.Endpoints.DeployModel(endpoint, request).Do()

	if err != nil {
		return fmt.Errorf("unable to deploy model: %v", err)
	}

	return poll(ctx, aiPlatformService, op)
}

// undeployNoTrafficModels fetches the Vertex AI endpoint and und-deploys all the models that have no traffic routed to them.
func undeployNoTrafficModels(ctx context.Context, aiPlatformService *aiplatform.Service, endpointName string) error {
	endpoint, err := aiPlatformService.Projects.Locations.Endpoints.Get(endpointName).Do()
	if err != nil {
		return fmt.Errorf("unable to fetch endpoint where model was deployed: %v", err)
	}

	var modelsToUndeploy = map[string]bool{}
	for _, dm := range endpoint.DeployedModels {
		modelsToUndeploy[dm.Id] = true
	}

	for id, split := range endpoint.TrafficSplit {

		// model does not get un-deployed if its configured to receive  traffic
		if split != 0 {
			delete(modelsToUndeploy, id)
		}
	}

	undeployedCount := 0
	err = nil
	var lros []*aiplatform.GoogleLongrunningOperation
	for id, _ := range modelsToUndeploy {
		undeployRequest := &aiplatform.GoogleCloudAiplatformV1UndeployModelRequest{DeployedModelId: id}
		lro, lroErr := aiPlatformService.Projects.Locations.Endpoints.UndeployModel(endpointName, undeployRequest).Do()
		if err != nil {
			fmt.Printf("error undeploying model: %v\n", err)
			err = lroErr
			undeployedCount += 1
		} else {
			lros = append(lros, lro)
		}
	}

	for pollErr := range pollChan(ctx, aiPlatformService, lros...) {
		if pollErr != nil {
			fmt.Printf("Error in undeploy model operation: %v", err)
			err = pollErr
		}
	}
	return err
}
