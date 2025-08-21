// Package cdenv contains Cloud Deploy environment variable keys.
package cdenv

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
