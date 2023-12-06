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

// Environment variable keys whose values determine the behavior of the Terraform deployer.
// These are set as deploy parameters in Cloud Deploy.
const (
	backendBucketEnvKey    = "CLOUD_DEPLOY_customTarget_tfBackendBucket"
	backendPrefixEnvKey    = "CLOUD_DEPLOY_customTarget_tfBackendPrefix"
	configPathEnvKey       = "CLOUD_DEPLOY_customTarget_tfConfigurationPath"
	variablePathEnvKey     = "CLOUD_DEPLOY_customTarget_tfVariablePath"
	enableRenderPlanEnvKey = "CLOUD_DEPLOY_customTarget_tfEnableRenderPlan"
	lockTimeoutEnvKey      = "CLOUD_DEPLOY_customTarget_tfLockTimeout"
	applyParallelismEnvKey = "CLOUD_DEPLOY_customTarget_tfApplyParallelism"
)

// params contains the deploy parameter values passed into the execution environment.
type params struct {
	// Name of the Cloud Storage bucket used to store the Terraform state.
	backendBucket string
	// Prefix to use for the Cloud Storage objects that represent the Terraform state.
	backendPrefix string
	// Path to the Terraform configuration in the Cloud Deploy Release archive. If not
	// provided then defaults to the root directory of the archive.
	configPath string
	// Path to a variable file relative to the Terraform configuration directory.
	variablePath string
	// Whether to generate a Terraform plan at render time for informational purposes,
	// i.e. provided in the Cloud Deploy Release inspector. Not used at apply time.
	enableRenderPlan bool
	// Duration to retry a state lock, when unset Terraform defaults to 0s.
	lockTimeout string
	// Parallelism to set when performing terraform apply, when unset Terraform
	// defaults to 10.
	applyParallelism int
}

// determineParams returns the params provided in the execution environment via environment variables.
func determineParams() (*params, error) {
	backendBucket := os.Getenv(backendBucketEnvKey)
	if len(backendBucket) == 0 {
		return nil, fmt.Errorf("parameter %q is required", backendBucketEnvKey)
	}
	backendPrefix := os.Getenv(backendPrefixEnvKey)
	if len(backendPrefix) == 0 {
		return nil, fmt.Errorf("parameter %q is required", backendPrefixEnvKey)
	}

	enablePlan := false
	ep, ok := os.LookupEnv(enableRenderPlanEnvKey)
	if ok {
		var err error
		enablePlan, err = strconv.ParseBool(ep)
		if err != nil {
			return nil, fmt.Errorf("failed to parse parameter %q: %v", enableRenderPlanEnvKey, err)
		}
	}

	var applyParallelism int
	ap, ok := os.LookupEnv(applyParallelismEnvKey)
	if ok {
		var err error
		applyParallelism, err = strconv.Atoi(ap)
		if err != nil {
			return nil, fmt.Errorf("failed to parse parameter %q: %v", applyParallelismEnvKey, err)
		}
	}

	return &params{
		backendBucket:    backendBucket,
		backendPrefix:    backendPrefix,
		configPath:       os.Getenv(configPathEnvKey),
		variablePath:     os.Getenv(variablePathEnvKey),
		enableRenderPlan: enablePlan,
		lockTimeout:      os.Getenv(lockTimeoutEnvKey),
		applyParallelism: applyParallelism,
	}, nil
}
