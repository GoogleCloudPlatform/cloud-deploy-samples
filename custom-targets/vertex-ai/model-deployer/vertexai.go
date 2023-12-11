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

func determinePreviousModel(service *aiplatform.Service, endpointName, model string) (string, error) {
	endpoint, err := service.Projects.Locations.Endpoints.Get(endpointName).Do()
	if err != nil {
		return "", fmt.Errorf("unable to fetch endpoint: %v", err)
	}

	deployedModels := map[string]*aiplatform.GoogleCloudAiplatformV1DeployedModel{}

	for _, dm := range endpoint.DeployedModels {
		modelNameWithVersion := resolveDeployedModelNameWithVersion(dm)
		deployedModels[modelNameWithVersion] = dm
	}

	delete(deployedModels, model)

	if len(deployedModels) != 1 {
		return "", fmt.Errorf("unable to resolve previous deployed model to canary against. Not including the current model to be deployed, the endpoint has %d deployed models but expected only one", len(deployedModels))
	}

	firstModel := []*aiplatform.GoogleCloudAiplatformV1DeployedModel{}

	for _, dm := range deployedModels {
		firstModel = append(firstModel, dm)
	}
	return firstModel[0].Id, nil
}

func resolveDeployedModelNameWithVersion(deployedModel *aiplatform.GoogleCloudAiplatformV1DeployedModel) string {
	if strings.Contains(deployedModel.Model, "@") {
		return deployedModel.Model
	}
	return fmt.Sprintf("%s@%s", deployedModel.Model, deployedModel.ModelVersionId)
}

func resolveModelWithVersion(model *aiplatform.GoogleCloudAiplatformV1Model) string {
	if strings.Contains(model.Name, "@") {
		return model.Name
	}
	return fmt.Sprintf("%s@%s", model.Name, model.VersionId)
}

// extracts the region from the model region name
func fetchRegionFromModel(modelName string) (string, error) {
	matches := modelRegex.FindStringSubmatch(modelName)
	if len(matches) == 0 {
		return "", fmt.Errorf("unable to parse model name")
	}

	return matches[2], nil
}

// extracts the region from the endpoint resource name
func fetchRegionFromEndpoint(endpointName string) (string, error) {
	matches := endpointRegex.FindStringSubmatch(endpointName)
	if len(matches) == 0 {
		return "", fmt.Errorf("unable to parse endpoint name")
	}

	return matches[2], nil
}

// newAIPlatformService generates a Service that can make API calls in the specified region
func newAIPlatformService(ctx context.Context, region string) (*aiplatform.Service, error) {
	endPointOption := option.WithEndpoint(fmt.Sprintf("%s-aiplatform.googleapis.com", region))
	regionalService, err := aiplatform.NewService(ctx, endPointOption)
	if err != nil {
		return nil, fmt.Errorf("unable to authenticate")
	}

	return regionalService, nil
}

func fetchModel(service *aiplatform.Service, modelName string) (*aiplatform.GoogleCloudAiplatformV1Model, error) {
	model, err := service.Projects.Locations.Models.Get(modelName).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %v", err)
	}
	return model, nil
}

func validateRequest(modelNameFromDeployParameter, endpointName string, minReplicaCountParameter int64, deployedModel *aiplatform.GoogleCloudAiplatformV1DeployedModel) error {
	modelRegion, err := fetchRegionFromModel(modelNameFromDeployParameter)
	if err != nil {
		return fmt.Errorf("unable to parse region from model: %v", err)
	}

	endpointRegion, err := fetchRegionFromEndpoint(endpointName)
	if err != nil {
		return fmt.Errorf("unable to parse region from endpoint: %v", err)
	}

	if endpointRegion != modelRegion {
		return fmt.Errorf("The model to be deployed must be in the same region as the endpoint. Copy the model to the region the  endpoint is located, or make an endpoint in the same region as the model")
	}

	if err = verifyModelNameNotDefinedInConfig(deployedModel); err != nil {
		return err
	}

	if err = verifyMinReplicaCountHasNoConflicts(deployedModel, minReplicaCountParameter); err != nil {
		return err
	}

	return nil
}

func verifyMinReplicaCountHasNoConflicts(deployedModel *aiplatform.GoogleCloudAiplatformV1DeployedModel, deployParameterValue int64) error {

	configValue := minReplicaCountFromConfig(deployedModel)

	// checks if minReplicaCount is not defined either in deploy parameter or config file
	if configValue == deployParameterValue {
		if configValue == 0 {
			return fmt.Errorf("minReplicaCount must either be defined in the config file or provided to the render operation through a deploy parameter using 'vertexAIMinReplicaCount' key")
		}
	}

	// only other valid format is if either but not both are 0
	if configValue == 0 || deployParameterValue == 0 {
		return nil
	}
	return fmt.Errorf("the minReplicaCount parameter is defined in both the provided config file and as a deploy parameter and both values differ from each other, please define minReplicaCount in the config file or as a deploy-parameter")
}

func minReplicaCountFromConfig(deployedModel *aiplatform.GoogleCloudAiplatformV1DeployedModel) int64 {
	if deployedModel.DedicatedResources != nil {
		return deployedModel.DedicatedResources.MinReplicaCount
	}
	return 0
}
func verifyModelNameNotDefinedInConfig(deployedModel *aiplatform.GoogleCloudAiplatformV1DeployedModel) error {

	if deployedModel.Model != "" {
		return fmt.Errorf("model to deployed must be supplied as a deploy parameter and not in the config file")
	}

	if deployedModel.ModelVersionId != "" {
		return fmt.Errorf("the model version id to deploy must be supplied as part of the vertexAIModel deployparamater containing the model to be deployed")
	}

	return nil

}
