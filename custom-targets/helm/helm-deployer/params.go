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
// Cloud Deploy transforms a deploy parameter "customTarget/helmGKECluster" into an
// environment variable of the form "CLOUD_DEPLOY_customTarget_helmGKECluster".
const (
	gkeClusterEnvkey       = "CLOUD_DEPLOY_customTarget_helmGKECluster"
	configPathEnvKey       = "CLOUD_DEPLOY_customTarget_helmConfigurationPath"
	namespaceEnvKey        = "CLOUD_DEPLOY_customTarget_helmNamespace"
	templateLookupEnvKey   = "CLOUD_DEPLOY_customTarget_helmTemplateLookup"
	templateValidateEnvKey = "CLOUD_DEPLOY_customTarget_helmTemplateValidate"
	upgradeTimeoutEnvKey   = "CLOUD_DEPLOY_customTarget_helmUpgradeTimeout"
)

// params contains the deploy parameter values passed into the execution environment.
type params struct {
	// Name of the GKE cluster.
	gkeCluster string
	// Path to the helm chart in the Cloud Deploy release archive. If not provided then
	// defaults to "mychart" in the root directory of the archive.
	configPath string
	// Namespace scope of the request.
	namespace string
	// Whether to handle lookup functions when performing helm template for the informational
	// release manifest, requires connecting to the cluster at render time.
	templateLookup bool
	// Whether to validate the manifest produced by helm template against the cluster,
	// requires connecting to the cluster at render time.
	templateValidate bool
	// Timeout duration when performing helm upgrade.
	upgradeTimeout string
}

// determineParams returns the params provided in the execution environment via environment variables.
func determineParams() (*params, error) {
	cluster := os.Getenv(gkeClusterEnvkey)
	if len(cluster) == 0 {
		return nil, fmt.Errorf("parameter %q is required", gkeClusterEnvkey)
	}

	templateLookup := false
	tl, ok := os.LookupEnv(templateLookupEnvKey)
	if ok {
		var err error
		templateLookup, err = strconv.ParseBool(tl)
		if err != nil {
			return nil, fmt.Errorf("failed to parse parameter %q: %v", templateLookupEnvKey, err)
		}
	}

	templateValidate := false
	tv, ok := os.LookupEnv(templateValidateEnvKey)
	if ok {
		var err error
		templateLookup, err = strconv.ParseBool(tv)
		if err != nil {
			return nil, fmt.Errorf("failed to parse parameter %q: %v", templateValidateEnvKey, err)
		}
	}

	return &params{
		gkeCluster:       cluster,
		configPath:       os.Getenv(configPathEnvKey),
		namespace:        os.Getenv(namespaceEnvKey),
		templateLookup:   templateLookup,
		templateValidate: templateValidate,
		upgradeTimeout:   os.Getenv(upgradeTimeoutEnvKey),
	}, nil
}
