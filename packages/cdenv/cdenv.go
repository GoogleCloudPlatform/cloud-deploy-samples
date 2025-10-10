// Package cdenv contains Cloud Deploy environment variable keys and utility functions for environment
// variables.
package cdenv

import (
	"fmt"
	"strings"
)

// Cloud Deploy environment variable keys.
const (
	RequestTypeEnvKey = "CLOUD_DEPLOY_REQUEST_TYPE"
	FeaturesEnvKey    = "CLOUD_DEPLOY_FEATURES"
	// ProjectEnvKey contains the project number of the Cloud Deploy resource.
	ProjectEnvKey = "CLOUD_DEPLOY_PROJECT"
	// ProjectIDEnvKey contains the project ID of the Cloud Deploy resource.
	ProjectIDEnvKey   = "CLOUD_DEPLOY_PROJECT_ID"
	LocationEnvKey    = "CLOUD_DEPLOY_LOCATION"
	PipelineEnvKey    = "CLOUD_DEPLOY_DELIVERY_PIPELINE"
	ReleaseEnvKey     = "CLOUD_DEPLOY_RELEASE"
	RolloutEnvKey     = "CLOUD_DEPLOY_ROLLOUT"
	TargetEnvKey      = "CLOUD_DEPLOY_TARGET"
	PhaseEnvKey       = "CLOUD_DEPLOY_PHASE"
	PercentageEnvKey  = "CLOUD_DEPLOY_PERCENTAGE_DEPLOY"
	StorageTypeEnvKey = "CLOUD_DEPLOY_STORAGE_TYPE"
	// InputGCSEnvKey contains the GCS URI where the users prerendered artifacts are located.
	InputGCSEnvKey = "CLOUD_DEPLOY_INPUT_GCS_PATH"
	// OutputGCSEnvKey is provided by Cloud Deploy. It is the GCS URI to use to upload a results
	// file.
	OutputGCSEnvKey = "CLOUD_DEPLOY_OUTPUT_GCS_PATH"
	// SkaffoldGCSEnvKey contains the GCR URI where the Skaffold configuration was uploaded to.
	SkaffoldGCSEnvKey = "CLOUD_DEPLOY_SKAFFOLD_GCS_PATH"
	// ManifestGCSEnvKey contains the path to the manifest file relative to the rendered output uri.
	ManifestGCSEnvKey        = "CLOUD_DEPLOY_MANIFEST_GCS_PATH"
	WorkloadTypeEnvKey       = "CLOUD_DEPLOY_WORKLOAD_TYPE"
	CloudBuildServiceAccount = "CLOUD_DEPLOY_WP_CB_ServiceAccount"
	CloudBuildWorkerPool     = "CLOUD_DEPLOY_WP_CB_WorkerPool"
)

// CheckDuplicates expects environment variables in the k=v format. It
// converts the environment string slice to a map and checks for duplicates
// and malformed entries.
func CheckDuplicates(environ []string) (map[string]string, error) {
	envMap := make(map[string]string)

	if len(environ) == 0 {
		return nil, fmt.Errorf("no environment variables found")
	}

	for _, envVar := range environ {
		pair := strings.SplitN(envVar, "=", 2)
		if len(pair) != 2 {
			return nil, fmt.Errorf("incorrect env variable format - expected k=v")
		}

		key := pair[0]
		value := pair[1]
		if key == "" {
			return nil, fmt.Errorf("empty environment variable key")
		}

		if value == "" {
			return nil, fmt.Errorf("empty environment variable value")
		}

		if _, exists := envMap[strings.ToLower(key)]; exists {
			return nil, fmt.Errorf("duplicate environment variable key: %s", key)
		}
		envMap[strings.ToLower(key)] = value
	}
	return envMap, nil
}
