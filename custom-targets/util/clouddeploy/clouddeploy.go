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

// Package clouddeploy provides functionality for working with custom render
// and custom deploy requests and results.
package clouddeploy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/cloud-deploy-samples/packages/cdenv"
	"github.com/GoogleCloudPlatform/cloud-deploy-samples/packages/gcs"
	"github.com/mholt/archiver/v3"
)

// GitCommit SHA to be set during build time of the binary.
var GitCommit = "unknown"

const (

	// cloudDeployEnvVarPrefix is the prefix for cloud deploy environment variables.
	cloudDeployEnvVarPrefix = "CLOUD_DEPLOY_"

	// cloudDeployCustomTargetEnvVarPrefix is the prefix for environment variables that represent deploy parameters configured in the "customTarget/" namespace.
	cloudDeployCustomTargetEnvVarPrefix = "CLOUD_DEPLOY_customTarget_"
)

// RenderRequest contains the Cloud Deploy values passed into the execution environment for a render operation.
type RenderRequest struct {
	// Cloud Deploy project.
	Project string
	// Cloud Deploy location.
	Location string
	// Cloud Deploy delivery pipeline.
	Pipeline string
	// Cloud Deploy release.
	Release string
	// Cloud Deploy target for this render.
	Target string
	// Cloud Deploy rollout phase.
	Phase string
	// Percentage deployment requested.
	Percentage int
	// The storage type for inputs and outputs. Currently only "GCS" is supported.
	StorageType string
	// Cloud Storage path to the tar.gz archive provided at the time of release creation in Cloud Deploy.
	// Example: gs://my-bucket/dir/subdir/source.tar.gz
	InputGCSPath string
	// Cloud Storage path where the outputs for the deploy are expected to be stored by Cloud Deploy. This
	// includes the results.json file and any rendered artifacts that need to be accessible at
	// deploy time.
	// Example: gs/my-bucket/dir/render-subdir/custom-output
	OutputGCSPath string
	// The workload type for the execution environment. Currently only "CB" is supported.
	WorkloadType string
	// Information about the Cloud Build workload. Only present when WorkloadType is "CB".
	WorkloadCBInfo CloudBuildWorkload
}

// CloudBuildWorkload provides workload execution context when running in Cloud Build.
type CloudBuildWorkload struct {
	// Service Account used by the Cloud Build.
	ServiceAccount string
	// Worker Pool the Build is running in. Empty if using Cloud Build's default pool.
	WorkerPool string
}

// RenderResult represents the json data expected in the results file by Cloud Deploy for a render operation.
type RenderResult struct {
	ResultStatus   RenderStatus      `json:"resultStatus"`
	ManifestFile   string            `json:"manifestFile"`
	FailureMessage string            `json:"failureMessage,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// RenderStatus represents the valid result status for a render request.
type RenderStatus string

const (
	// RenderSucceeded is the render succeeded status.
	RenderSucceeded RenderStatus = "SUCCEEDED"
	// RenderFailed is the render failed status.
	RenderFailed RenderStatus = "FAILED"
	// RenderNotSupported is the render not supported status.
	RenderNotSupported RenderStatus = "NOT_SUPPORTED"
)

// Cloud Deploy known result metadata keys.
const (
	CustomTargetSourceMetadataKey    = "custom-target-source"
	CustomTargetSourceSHAMetadataKey = "custom-target-source-commit-sha"
)

// DownloadAndUnarchiveInput downloads the release archive and unarchives it to the provided path.
// Returns the Cloud Storage URI of the downloaded archive.
func (r *RenderRequest) DownloadAndUnarchiveInput(ctx context.Context, gcsClient *storage.Client, localArchivePath, localUnarchivePath string) (string, error) {
	// For render the input gcs path is the path to the source archive.
	uri := r.InputGCSPath
	out, err := gcs.Download(ctx, gcsClient, uri, localArchivePath)
	if err != nil {
		return "", err
	}
	// Unarchive the tarball downloaded from GCS into the provided unarchive path.
	if err := archiver.NewTarGz().Unarchive(out.Name(), localUnarchivePath); err != nil {
		return "", fmt.Errorf("unable to unarchive tarball from %q: %v", uri, err)
	}
	return uri, nil
}

// UploadArtifact uploads the provided content as a rendered artifact. The objectSuffix must be provided
// to determine the Cloud Storage URI to use for the object, the URI is returned.
func (r *RenderRequest) UploadArtifact(ctx context.Context, gcsClient *storage.Client, objectSuffix string, content *gcs.UploadContent) (string, error) {
	if len(objectSuffix) == 0 {
		return "", fmt.Errorf("objectSuffix must be provided to upload a render artifact")
	}
	// For render the output gcs path is the path to a Cloud Storage directory.
	uri := fmt.Sprintf("%s/%s", r.OutputGCSPath, objectSuffix)
	if err := gcs.Upload(ctx, gcsClient, uri, content); err != nil {
		return "", err
	}
	return uri, nil
}

// UploadResult uploads the provided render result to the Cloud Storage path where Cloud Deploy expects it.
// Returns the Cloud Storage URI of the uploaded result.
func (r *RenderRequest) UploadResult(ctx context.Context, gcsClient *storage.Client, renderResult *RenderResult) (string, error) {
	uri := fmt.Sprintf("%s/%s", r.OutputGCSPath, gcs.ResultObjectSuffix)
	res, err := json.Marshal(renderResult)
	if err != nil {
		return "", fmt.Errorf("error marshalling render result: %v", err)
	}
	if err := gcs.Upload(ctx, gcsClient, uri, &gcs.UploadContent{Data: res}); err != nil {
		return "", err
	}
	return uri, nil
}

// DeployRequest contains the Cloud Deploy values passed into the execution environment for a deploy operation.
type DeployRequest struct {
	// Cloud Deploy project.
	Project string
	// Cloud Deploy location.
	Location string
	// Cloud Deploy delivery pipeline.
	Pipeline string
	// Cloud Deploy release.
	Release string
	// Cloud Deploy rollout.
	Rollout string
	// Cloud Deploy target for this deploy.
	Target string
	// Cloud Deploy rollout phase.
	Phase string
	// Percentage deployment requested.
	Percentage int
	// The storage type for inputs and outputs. Currently only GCS is supported.
	StorageType string
	// Cloud Storage path where the inputs for the deploy are stored. This is equivalent to the output GCS
	// path for the renderer. If Cloud Deploy performed the render via skaffold instead of this
	// image then the input is the manifest path instead.
	// Example: gs://my-bucket/render/dir/subdir/custom-output
	InputGCSPath string
	// Cloud Storage path for the skaffold config file.
	// Example: gs://my-bucket/dir/render-subdir/skaffold.yaml
	SkaffoldGCSPath string
	// Cloud Storage path for the manifest file. This is either the manifest provided by this images render
	// or the manifest from a skaffold render if the default render was configured.
	// Example: gs//my-bucket/dir/render-subdir/manifest.yaml
	ManifestGCSPath string
	// Cloud Storage path where the outputs for the deploy are expected to be stored by Cloud Deploy. This
	// includes the results.json file and any deploy artifacts Cloud Deploy should populate in its
	// resources.
	// Example: gs/my-bucket/dir/deploy-subdir/custom-output
	OutputGCSPath string
	// The workload type for the execution environment. Currently only "CB" is supported.
	WorkloadType string
	// Information about the Cloud Build workload. Only present when WorkloadType is "CB".
	WorkloadCBInfo CloudBuildWorkload
}

// DeployResult represents the json data expected in the results file by Cloud Deploy for a deploy operation.
type DeployResult struct {
	ResultStatus   DeployStatus      `json:"resultStatus"`
	ArtifactFiles  []string          `json:"artifactFiles,omitempty"`
	FailureMessage string            `json:"failureMessage,omitempty"`
	SkipMessage    string            `json:"skipMessage,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// DeployStatus represents the valid result status for a deploy request.
type DeployStatus string

const (
	// DeploySucceeded is the deploy succeeded status.
	DeploySucceeded DeployStatus = "SUCCEEDED"
	// DeployFailed is the deploy failed status.
	DeployFailed DeployStatus = "FAILED"
	// DeploySkipped is the deploy skipped status.
	DeploySkipped DeployStatus = "SKIPPED"
	// DeployNotSupported is the deploy not supported status.
	DeployNotSupported DeployStatus = "NOT_SUPPORTED"
)

// DownloadInput downloads the deploy input with the specified objectSuffix from Cloud Storage to the provided local path.
// Returns the Cloud Storage URI of the downloaded input.
func (d *DeployRequest) DownloadInput(ctx context.Context, gcsClient *storage.Client, objectSuffix, localPath string) (string, error) {
	// For deploy the input gcs path is a path to a GCS directory. Need the suffix used when uploading at render
	// time to determine the object to download.
	uri := fmt.Sprintf("%s/%s", d.InputGCSPath, objectSuffix)
	_, err := gcs.Download(ctx, gcsClient, uri, localPath)
	if err != nil {
		return "", err
	}
	return uri, nil
}

// DownloadManifest downloads the manifest to the provided local path. Returns the Cloud Storage URI of the downloaded manifest.
func (d *DeployRequest) DownloadManifest(ctx context.Context, gcsClient *storage.Client, localPath string) (string, error) {
	// The manifest gcs path is the path to the manifest file provided at render time.
	uri := d.ManifestGCSPath
	if _, err := gcs.Download(ctx, gcsClient, uri, localPath); err != nil {
		return "", err
	}
	return uri, nil
}

// UploadArtifact uploads the provided content as a deploy artifact. The objectSuffix must be provided
// to determine the Cloud Storage URI to use for the object, the URI is returned.
func (d *DeployRequest) UploadArtifact(ctx context.Context, gcsClient *storage.Client, objectSuffix string, content *gcs.UploadContent) (string, error) {
	if len(objectSuffix) == 0 {
		return "", fmt.Errorf("objectSuffix must be provided to upload a deploy artifact")
	}
	// For deploy the output gcs path is the path to a Cloud Storage directory.
	uri := fmt.Sprintf("%s/%s", d.OutputGCSPath, objectSuffix)
	if err := gcs.Upload(ctx, gcsClient, uri, content); err != nil {
		return "", err
	}
	return uri, nil
}

// UploadResult uploads the provided deploy result to the Cloud Storage path where Cloud Deploy expects it.
// Returns the Cloud Storage URI of the uploaded result.
func (d *DeployRequest) UploadResult(ctx context.Context, gcsClient *storage.Client, deployResult *DeployResult) (string, error) {
	uri := fmt.Sprintf("%s/%s", d.OutputGCSPath, gcs.ResultObjectSuffix)
	res, err := json.Marshal(deployResult)
	if err != nil {
		return "", fmt.Errorf("error marshalling deploy result: %v", err)
	}
	if err := gcs.Upload(ctx, gcsClient, uri, &gcs.UploadContent{Data: res}); err != nil {
		return "", err
	}
	return uri, nil
}

// DetermineRequest determines the Cloud Deploy request based on the environment variables in the
// execution environment and returns either a RenderRequest or DeployRequest. If the request
// includes a feature that is not in provided supported features list then a NOT_SUPPORTED result
// is uploaded for Cloud Deploy and an error is returned.
func DetermineRequest(ctx context.Context, gcsClient *storage.Client, supportedFeatures []string) (any, error) {
	// Values present for render and deploy.
	project := os.Getenv(cdenv.ProjectEnvKey)
	location := os.Getenv(cdenv.LocationEnvKey)
	pipeline := os.Getenv(cdenv.PipelineEnvKey)
	release := os.Getenv(cdenv.ReleaseEnvKey)
	target := os.Getenv(cdenv.TargetEnvKey)
	phase := os.Getenv(cdenv.PhaseEnvKey)
	percentage, err := strconv.Atoi(os.Getenv(cdenv.PercentageEnvKey))
	if err != nil {
		return nil, fmt.Errorf("failed to parse %q", cdenv.PercentageEnvKey)
	}
	storageType := os.Getenv(cdenv.StorageTypeEnvKey)
	inputGCSPath := os.Getenv(cdenv.InputGCSEnvKey)
	outputGCSPath := os.Getenv(cdenv.OutputGCSEnvKey)

	workloadType := os.Getenv(cdenv.WorkloadTypeEnvKey)
	var cbWorkload CloudBuildWorkload
	if workloadType == "CB" {
		cbWorkload = CloudBuildWorkload{
			ServiceAccount: os.Getenv(cdenv.CloudBuildServiceAccount),
			WorkerPool:     os.Getenv(cdenv.CloudBuildWorkerPool),
		}
	}

	features := strings.FieldsFunc(os.Getenv(cdenv.FeaturesEnvKey), func(c rune) bool {
		return c == ','
	})

	reqType := os.Getenv(cdenv.RequestTypeEnvKey)
	switch reqType {
	case "RENDER":
		rr := &RenderRequest{
			Project:        project,
			Location:       location,
			Pipeline:       pipeline,
			Release:        release,
			Target:         target,
			Phase:          phase,
			Percentage:     percentage,
			StorageType:    storageType,
			InputGCSPath:   inputGCSPath,
			OutputGCSPath:  outputGCSPath,
			WorkloadType:   workloadType,
			WorkloadCBInfo: cbWorkload,
		}

		for _, f := range features {
			if !isFeatureSupported(supportedFeatures, f) {
				msg := fmt.Sprintf("feature %q is not supported", f)
				_, err := rr.UploadResult(ctx, gcsClient, &RenderResult{
					ResultStatus:   RenderNotSupported,
					FailureMessage: msg,
				})
				if err != nil {
					return nil, fmt.Errorf("error uploading render feature not supported results: %v", err)
				}
				return nil, errors.New(msg)
			}
		}
		return rr, nil

	case "DEPLOY":
		dr := &DeployRequest{
			Project:         project,
			Location:        location,
			Pipeline:        pipeline,
			Release:         release,
			Rollout:         os.Getenv(cdenv.RolloutEnvKey),
			Target:          target,
			Phase:           phase,
			Percentage:      percentage,
			StorageType:     storageType,
			InputGCSPath:    inputGCSPath,
			SkaffoldGCSPath: os.Getenv(cdenv.SkaffoldGCSEnvKey),
			ManifestGCSPath: os.Getenv(cdenv.ManifestGCSEnvKey),
			OutputGCSPath:   outputGCSPath,
			WorkloadType:    workloadType,
			WorkloadCBInfo:  cbWorkload,
		}

		for _, f := range features {
			if !isFeatureSupported(supportedFeatures, f) {
				msg := fmt.Sprintf("feature %q is not supported", f)
				_, err := dr.UploadResult(ctx, gcsClient, &DeployResult{
					ResultStatus:   DeployNotSupported,
					FailureMessage: msg,
				})
				if err != nil {
					return nil, fmt.Errorf("error uploading deploy feature not supported results: %v", err)
				}
				return nil, errors.New(msg)
			}
		}

		return dr, nil

	default:
		return nil, fmt.Errorf("received unexpected Cloud Deploy request type: %v", reqType)
	}
}

// isFeature supported returns whether the provided feature is in the list of supported features provided.
func isFeatureSupported(supportedFeatures []string, feature string) bool {
	for _, sf := range supportedFeatures {
		if sf == feature {
			return true
		}
	}
	return false
}

// isDeployParamAndKey determines if the provided env var key corresponds
// to a deploy parameter, if it is then it returns the deploy parameter key.
func isDeployParamAndKey(key string) (bool, string) {
	if strings.HasPrefix(key, cloudDeployCustomTargetEnvVarPrefix) {
		transformedKey := strings.TrimPrefix(key, cloudDeployCustomTargetEnvVarPrefix)
		transformedKey = fmt.Sprintf("customTarget/%s", transformedKey)
		return true, transformedKey
	} else if strings.HasPrefix(key, cloudDeployEnvVarPrefix) {
		return false, ""
	} else {
		return true, key
	}
}

// FetchDeployParameters returns a map of all the deploy parameters provided in the execution environment.
func FetchDeployParameters() map[string]string {
	params := map[string]string{}
	environs := os.Environ()
	for _, environ := range environs {
		segments := strings.Split(environ, "=")
		if validKey, transformedKey := isDeployParamAndKey(segments[0]); validKey {
			params[transformedKey] = segments[1]
		}
	}
	return params
}
