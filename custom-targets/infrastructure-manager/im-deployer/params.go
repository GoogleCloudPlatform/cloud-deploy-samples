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
	"fmt"
	"os"
	"strconv"
)

// Environment variable keys whose values determine the behavior of the Infrastructure Manager deployer.
// Cloud Deploy transforms a deploy parameter "customTarget/imProject" into an environment variable
// of the form "CLOUD_DEPLOY_customTarget_imProject".
const (
	imProjectEnvKey                = "CLOUD_DEPLOY_customTarget_imProject"
	imLocationEnvKey               = "CLOUD_DEPLOY_customTarget_imLocation"
	imDeploymentEnvKey             = "CLOUD_DEPLOY_customTarget_imDeployment"
	configPathEnvKey               = "CLOUD_DEPLOY_customTarget_imConfigurationPath"
	variablePathEnvKey             = "CLOUD_DEPLOY_customTarget_imVariablePath"
	imServiceAccountEnvKey         = "CLOUD_DEPLOY_customTarget_imServiceAccount"
	imWorkerPoolEnvKey             = "CLOUD_DEPLOY_customTarget_imWorkerPool"
	importExistingResourcesEnvKey  = "CLOUD_DEPLOY_customTarget_imImportExistingResources"
	disableCloudDeployLabelsEnvKey = "CLOUD_DEPLOY_customTarget_imDisableCloudDeployLabels"
	imVarEnvKeyPrefix              = "CLOUD_DEPLOY_customTarget_imVar_"
)

const (
	// The deploy parameter key prefix for variables.
	imVarDeployParamKeyPrefix = "customTarget/imVar_"
)

// params contains the deploy parameter values passed into the execution environment.
type params struct {
	// The project ID for the Infrastructure Manager Deployment.
	imProject string
	// The location for the Infrastructure Manager Deployment.
	imLocation string
	// The ID of the Infrastructure Manager Deployment responsible for managing the Terraform configuration.
	imDeployment string
	// Path to the Terraform configuration in the Cloud Deploy release archive. If not provided then
	// defaults to the root directory of the archive.
	configPath string
	// Path to a variable file relative to the Terraform configuration directory.
	variablePath string
	// Service account Infrastructure Manager uses when actuating resources. If not provided then defaults
	// to the service account provided by the Cloud Deploy workload context.
	imServiceAccount string
	// Worker Pool Infrastructure Manager uses when creating Cloud Builds. If not provided then defaults
	// to the worker pool provided by the Cloud Deploy workload context.
	imWorkerPool string
	// Whether Infrastructure Manager should automatically import existing resources into the Terraform
	// state and continue actuation.
	importExistingResources bool
	// Whether to disable the Cloud Deploy labels on the Infrastructure Manager Deployment resource.
	disableCloudDeployLabels bool
}

// determineParams returns the params provided in the execution environment via environment variables.
func determineParams() (*params, error) {
	imProject := os.Getenv(imProjectEnvKey)
	if len(imProject) == 0 {
		return nil, fmt.Errorf("parameter %q is required", imProjectEnvKey)
	}
	imLocation := os.Getenv(imLocationEnvKey)
	if len(imLocation) == 0 {
		return nil, fmt.Errorf("parameter %q is required", imLocationEnvKey)
	}
	imDeployment := os.Getenv(imDeploymentEnvKey)
	if len(imDeployment) == 0 {
		return nil, fmt.Errorf("parameter %q is required", imDeploymentEnvKey)
	}

	importRes := false
	ier, ok := os.LookupEnv(importExistingResourcesEnvKey)
	if ok {
		var err error
		importRes, err = strconv.ParseBool(ier)
		if err != nil {
			return nil, fmt.Errorf("failed to parse parameter %q: %v", importExistingResourcesEnvKey, err)
		}
	}
	disCDLabels := false
	dcdl, ok := os.LookupEnv(disableCloudDeployLabelsEnvKey)
	if ok {
		var err error
		disCDLabels, err = strconv.ParseBool(dcdl)
		if err != nil {
			return nil, fmt.Errorf("failed to parse parameter %q: %v", disableCloudDeployLabelsEnvKey, err)
		}
	}

	return &params{
		imProject:                imProject,
		imLocation:               imLocation,
		imDeployment:             imDeployment,
		imServiceAccount:         os.Getenv(imServiceAccountEnvKey),
		imWorkerPool:             os.Getenv(imWorkerPoolEnvKey),
		configPath:               os.Getenv(configPathEnvKey),
		variablePath:             os.Getenv(variablePathEnvKey),
		importExistingResources:  importRes,
		disableCloudDeployLabels: disCDLabels,
	}, nil
}

// deploymentName returns the name of the Infrastructure Manager Deployment.
func (p *params) deploymentName() string {
	return fmt.Sprintf("projects/%s/locations/%s/deployments/%s", p.imProject, p.imLocation, p.imDeployment)
}
