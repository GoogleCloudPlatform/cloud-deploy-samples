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
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/mholt/archiver/v3"
)

// GitCommit SHA to be set during build time of the binary.
var GitCommit = "unknown"

const (
	// cloudDeployPrefix is the prefix for environment variables containing information about the deployment
	cloudDeployPrefix = "CLOUD_DEPLOY_"

	// cloudDeployCustomTargetPrefix is the prefix for deploy parameters that are supported or required by the custom target.
	cloudDeployCustomTargetPrefix = "CLOUD_DEPLOY_customTarget_"
)

// Cloud Deploy environment variable keys.
const (
	RequestTypeEnvKey        = "CLOUD_DEPLOY_REQUEST_TYPE"
	FeaturesEnvKey           = "CLOUD_DEPLOY_FEATURES"
	ProjectEnvKey            = "CLOUD_DEPLOY_PROJECT"
	LocationEnvKey           = "CLOUD_DEPLOY_LOCATION"
	PipelineEnvKey           = "CLOUD_DEPLOY_DELIVERY_PIPELINE"
	ReleaseEnvKey            = "CLOUD_DEPLOY_RELEASE"
	RolloutEnvKey            = "CLOUD_DEPLOY_ROLLOUT"
	TargetEnvKey             = "CLOUD_DEPLOY_TARGET"
	PhaseEnvKey              = "CLOUD_DEPLOY_PHASE"
	PercentageEnvKey         = "CLOUD_DEPLOY_PERCENTAGE_DEPLOY"
	StorageTypeEnvKey        = "CLOUD_DEPLOY_STORAGE_TYPE"
	InputGCSEnvKey           = "CLOUD_DEPLOY_INPUT_GCS_PATH"
	OutputGCSEnvKey          = "CLOUD_DEPLOY_OUTPUT_GCS_PATH"
	SkaffoldGCSEnvKey        = "CLOUD_DEPLOY_SKAFFOLD_GCS_PATH"
	ManifestGCSEnvKey        = "CLOUD_DEPLOY_MANIFEST_GCS_PATH"
	WorkloadTypeEnvKey       = "CLOUD_DEPLOY_WORKLOAD_TYPE"
	CloudBuildServiceAccount = "CLOUD_DEPLOY_WP_CB_ServiceAccount"
	CloudBuildWorkerPool     = "CLOUD_DEPLOY_WP_CB_WorkerPool"
)

const (
	// The Cloud Storage object suffix for the expected results file.
	resultObjectSuffix = "results.json"
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
	RenderSucceeded    RenderStatus = "SUCCEEDED"
	RenderFailed       RenderStatus = "FAILED"
	RenderNotSupported RenderStatus = "NOT_SUPPORTED"
)

// Cloud Deploy known result metadata keys.
const (
	CustomTargetSourceMetadataKey = "custom-target-source"
)

// DownloadAndUnarchiveInput downloads the release archive and unarchives it to the provided path.
// Returns the Cloud Storage URI of the downloaded archive.
func (r *RenderRequest) DownloadAndUnarchiveInput(ctx context.Context, gcsClient *storage.Client, localArchivePath, localUnarchivePath string) (string, error) {
	// For render the input gcs path is the path to the source archive.
	uri := r.InputGCSPath
	out, err := downloadGCS(ctx, gcsClient, uri, localArchivePath)
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
func (r *RenderRequest) UploadArtifact(ctx context.Context, gcsClient *storage.Client, objectSuffix string, content *GCSUploadContent) (string, error) {
	if len(objectSuffix) == 0 {
		return "", fmt.Errorf("objectSuffix must be provided to upload a render artifact")
	}
	// For render the output gcs path is the path to a Cloud Storage directory.
	uri := fmt.Sprintf("%s/%s", r.OutputGCSPath, objectSuffix)
	if err := uploadGCS(ctx, gcsClient, uri, content); err != nil {
		return "", err
	}
	return uri, nil
}

// UploadResult uploads the provided render result to the Cloud Storage path where Cloud Deploy expects it.
// Returns the Cloud Storage URI of the uploaded result.
func (r *RenderRequest) UploadResult(ctx context.Context, gcsClient *storage.Client, renderResult *RenderResult) (string, error) {
	uri := fmt.Sprintf("%s/%s", r.OutputGCSPath, resultObjectSuffix)
	res, err := json.Marshal(renderResult)
	if err != nil {
		return "", fmt.Errorf("error marshalling render result: %v", err)
	}
	if err := uploadGCS(ctx, gcsClient, uri, &GCSUploadContent{Data: res}); err != nil {
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
	DeploySucceeded    DeployStatus = "SUCCEEDED"
	DeployFailed       DeployStatus = "FAILED"
	DeploySkipped      DeployStatus = "SKIPPED"
	DeployNotSupported DeployStatus = "NOT_SUPPORTED"
)

// DownloadInput downloads the deploy input with the specified objectSuffix from Cloud Storage to the provided local path.
// Returns the Cloud Storage URI of the downloaded input.
func (d *DeployRequest) DownloadInput(ctx context.Context, gcsClient *storage.Client, objectSuffix, localPath string) (string, error) {
	// For deploy the input gcs path is a path to a GCS directory. Need the suffix used when uploading at render
	// time to determine the object to download.
	uri := fmt.Sprintf("%s/%s", d.InputGCSPath, objectSuffix)
	_, err := downloadGCS(ctx, gcsClient, uri, localPath)
	if err != nil {
		return "", err
	}
	return uri, nil
}

// DownloadManifest downloads the manifest to the provided local path. Returns the Cloud Storage URI of the downloaded manifest.
func (d *DeployRequest) DownloadManifest(ctx context.Context, gcsClient *storage.Client, localPath string) (string, error) {
	// The manifest gcs path is the path to the manifest file provided at render time.
	uri := d.ManifestGCSPath
	if _, err := downloadGCS(ctx, gcsClient, uri, localPath); err != nil {
		return "", err
	}
	return uri, nil
}

// UploadArtifact uploads the provided content as a deploy artifact. The objectSuffix must be provided
// to determine the Cloud Storage URI to use for the object, the URI is returned.
func (d *DeployRequest) UploadArtifact(ctx context.Context, gcsClient *storage.Client, objectSuffix string, content *GCSUploadContent) (string, error) {
	if len(objectSuffix) == 0 {
		return "", fmt.Errorf("objectSuffix must be provided to upload a deploy artifact")
	}
	// For deploy the output gcs path is the path to a Cloud Storage directory.
	uri := fmt.Sprintf("%s/%s", d.OutputGCSPath, objectSuffix)
	if err := uploadGCS(ctx, gcsClient, uri, content); err != nil {
		return "", err
	}
	return uri, nil
}

// UploadResult uploads the provided deploy result to the Cloud Storage path where Cloud Deploy expects it.
// Returns the Cloud Storage URI of the uploaded result.
func (d *DeployRequest) UploadResult(ctx context.Context, gcsClient *storage.Client, deployResult *DeployResult) (string, error) {
	uri := fmt.Sprintf("%s/%s", d.OutputGCSPath, resultObjectSuffix)
	res, err := json.Marshal(deployResult)
	if err != nil {
		return "", fmt.Errorf("error marshalling deploy result: %v", err)
	}
	if err := uploadGCS(ctx, gcsClient, uri, &GCSUploadContent{Data: res}); err != nil {
		return "", err
	}
	return uri, nil
}

// DetermineRequest determines the Cloud Deploy request based on the environment variables in the
// execution environment and returns either a RenderRequest or DeployRequest. If the request
// includes a feature that is not in provided supported features list then a NOT_SUPPORTED result
// is uploaded for Cloud Deploy and an error is returned.
func DetermineRequest(ctx context.Context, gcsClient *storage.Client, supportedFeatures []string) (interface{}, error) {
	// Values present for render and deploy.
	project := os.Getenv(ProjectEnvKey)
	location := os.Getenv(LocationEnvKey)
	pipeline := os.Getenv(PipelineEnvKey)
	release := os.Getenv(ReleaseEnvKey)
	target := os.Getenv(TargetEnvKey)
	phase := os.Getenv(PhaseEnvKey)
	percentage, err := strconv.Atoi(os.Getenv(PercentageEnvKey))
	if err != nil {
		return nil, fmt.Errorf("failed to parse %q", PercentageEnvKey)
	}
	storageType := os.Getenv(StorageTypeEnvKey)
	inputGCSPath := os.Getenv(InputGCSEnvKey)
	outputGCSPath := os.Getenv(OutputGCSEnvKey)

	workloadType := os.Getenv(WorkloadTypeEnvKey)
	var cbWorkload CloudBuildWorkload
	if workloadType == "CB" {
		cbWorkload = CloudBuildWorkload{
			ServiceAccount: os.Getenv(CloudBuildServiceAccount),
			WorkerPool:     os.Getenv(CloudBuildWorkerPool),
		}
	}

	features := strings.FieldsFunc(os.Getenv(FeaturesEnvKey), func(c rune) bool {
		return c == ','
	})

	reqType := os.Getenv(RequestTypeEnvKey)
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
				return nil, fmt.Errorf(msg)
			}
		}
		return rr, nil

	case "DEPLOY":
		dr := &DeployRequest{
			Project:         project,
			Location:        location,
			Pipeline:        pipeline,
			Release:         release,
			Rollout:         os.Getenv(RolloutEnvKey),
			Target:          target,
			Phase:           phase,
			Percentage:      percentage,
			StorageType:     storageType,
			InputGCSPath:    inputGCSPath,
			SkaffoldGCSPath: os.Getenv(SkaffoldGCSEnvKey),
			ManifestGCSPath: os.Getenv(ManifestGCSEnvKey),
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
				return nil, fmt.Errorf(msg)
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

// downloadGCS downloads the Cloud Storage object for the specified URI to the provided local path.
func downloadGCS(ctx context.Context, gcsClient *storage.Client, gcsURI, localPath string) (*os.File, error) {
	gcsObj, err := parseGCSURI(gcsURI)
	if err != nil {
		return nil, err
	}
	r, err := gcsClient.Bucket(gcsObj.bucket).Object(gcsObj.name).NewReader(ctx)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	if err := os.MkdirAll(filepath.Dir(localPath), os.ModePerm); err != nil {
		return nil, err
	}
	file, err := os.Create(localPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	if _, err := io.Copy(file, r); err != nil {
		return nil, err
	}
	return file, nil
}

// GCSUploadContent is used as a parameter for the various GCS upload functions that points
// to the source of the content to upload.
type GCSUploadContent struct {
	// Content is this byte array.
	Data []byte
	// Content is in the file at this local path.
	LocalPath string
}

// uploadGCS uploads the provided content to the specified Cloud Storage URI.
func uploadGCS(ctx context.Context, gcsClient *storage.Client, gcsURI string, content *GCSUploadContent) error {
	// Determine the source of the content to upload.
	var contentData []byte
	switch {
	case len(content.Data) != 0:
		contentData = content.Data
	case len(content.LocalPath) != 0:
		var err error
		contentData, err = os.ReadFile(content.LocalPath)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unable to determine the content to upload to GCS")
	}

	gcsObjURI, err := parseGCSURI(gcsURI)
	if err != nil {
		return err
	}
	w := gcsClient.Bucket(gcsObjURI.bucket).Object(gcsObjURI.name).NewWriter(ctx)
	if _, err := w.Write(contentData); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return nil
}

// gcsObjectURI is used to split the object Cloud Storage URI into the bucket and name.
type gcsObjectURI struct {
	// bucket the GCS object is in.
	bucket string
	// name of the GCS object.
	name string
}

// parseGCSURI parses the Cloud Storage URI and returns the corresponding gcsObjectURI.
func parseGCSURI(uri string) (gcsObjectURI, error) {
	var obj gcsObjectURI
	u, err := url.Parse(uri)
	if err != nil {
		return gcsObjectURI{}, fmt.Errorf("cannot parse URI %q: %w", uri, err)
	}
	if u.Scheme != "gs" {
		return gcsObjectURI{}, fmt.Errorf("URI scheme is %q, must be 'gs'", u.Scheme)
	}
	if u.Host == "" {
		return gcsObjectURI{}, errors.New("bucket name is empty")
	}
	obj.bucket = u.Host
	obj.name = strings.TrimLeft(u.Path, "/")
	if obj.name == "" {
		return gcsObjectURI{}, errors.New("object name is empty")
	}
	return obj, nil
}

// transformAndValidateEnvkey checks if the environment variable is a valid deploy parameter
// and transforms the environment variable key back to the original format.
func transformAndValidateEnvkey(key string) (bool, string) {
	if strings.HasPrefix(key, cloudDeployCustomTargetPrefix) {
		transformedKey := strings.TrimPrefix(key, cloudDeployCustomTargetPrefix)
		transformedKey = fmt.Sprintf("customTarget/%s", transformedKey)
		return true, transformedKey
	} else if strings.HasPrefix(key, cloudDeployPrefix) {
		return false, ""
	} else {
		return true, key
	}
}

// FetchCloudDeployParameters returns a  map of all environment variables and keys
// that can be used in template parameterization.
func FetchCloudDeployParameters() map[string]string {
	params := map[string]string{}
	environs := os.Environ()
	for _, environ := range environs {
		segments := strings.Split(environ, "=")
		if validKey, transformedKey := transformAndValidateEnvkey(segments[0]); validKey {
			params[transformedKey] = segments[1]
		}
	}
	return params
}
