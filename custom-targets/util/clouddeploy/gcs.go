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
	"strings"

	"cloud.google.com/go/storage"
	"github.com/mholt/archiver/v3"
)

const (
	// The object suffix to use when uploading render or deploy results to GCS.
	resultObjectSuffix = "results.json"
)

// DownloadGCSAndUnarchiveRenderInput downloads the render input archive and unarchives it to the provided local paths.
// Returns the GCS URI of the downloaded archive.
func DownloadGCSAndUnarchiveRenderInput(ctx context.Context, gcsClient *storage.Client, renderRequest *RenderRequest, localArchivePath, localUnarchivePath string) (string, error) {
	// For render the input gcs path is the path to the source archive.
	uri := renderRequest.InputGCSPath
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

// DownloadGCSDeployInput downloads the deploy input with the specified objectSuffix to the provided local path.
// Returns the GCS URI of the downloaded input.
func DownloadGCSDeployInput(ctx context.Context, gcsClient *storage.Client, deployRequest *DeployRequest, objectSuffix, localPath string) (string, error) {
	// For deploy the input gcs path is a path to a GCS directory. Need the suffix used when uploading at render
	// time to determine the object to download.
	uri := fmt.Sprintf("%s/%s", deployRequest.InputGCSPath, objectSuffix)
	_, err := downloadGCS(ctx, gcsClient, uri, localPath)
	if err != nil {
		return "", err
	}
	return uri, nil
}

// DownloadGCSDeployManifest downloads the deploy manifest to the provided local path. Returns the GCS URI of the downloaded manifest.
func DownloadGCSDeployManifest(ctx context.Context, gcsClient *storage.Client, deployRequest *DeployRequest, localPath string) (string, error) {
	// The manifest gcs path is the path to the manifest file provided at render time.
	uri := deployRequest.ManifestGCSPath
	if _, err := downloadGCS(ctx, gcsClient, uri, localPath); err != nil {
		return "", err
	}
	return uri, nil
}

// GCSUploadContent is used as a parameter for the various GCS upload functions that points
// to the source of the content to upload.
type GCSUploadContent struct {
	// Content is this byte array.
	Data []byte
	// Content is in the file at this local path.
	LocalPath string
}

// UploadGCSRenderArtifact uploads the provided content as a rendered artifact. The objectSuffix must be provided
// to determine the URI to use for the GCS object, the URI is returned.
func UploadGCSRenderArtifact(ctx context.Context, gcsClient *storage.Client, renderRequest *RenderRequest, objectSuffix string, content *GCSUploadContent) (string, error) {
	if len(objectSuffix) == 0 {
		return "", fmt.Errorf("objectSuffix must be provided to upload a render artifact")
	}
	// For render the output gcs path is the path to a GCS directory.
	uri := fmt.Sprintf("%s/%s", renderRequest.OutputGCSPath, objectSuffix)
	if err := uploadGCS(ctx, gcsClient, uri, content); err != nil {
		return "", err
	}
	return uri, nil
}

// UploadGCSRenderResult uploads the provided render result to the GCS path expected by Cloud Deploy. Returns the GCS URI
// of the uploaded result.
func UploadGCSRenderResult(ctx context.Context, gcsClient *storage.Client, renderRequest *RenderRequest, renderResult *RenderResult) (string, error) {
	uri := fmt.Sprintf("%s/%s", renderRequest.OutputGCSPath, resultObjectSuffix)
	res, err := json.Marshal(renderResult)
	if err != nil {
		return "", fmt.Errorf("error marshalling render result: %v", err)
	}
	if err := uploadGCS(ctx, gcsClient, uri, &GCSUploadContent{Data: res}); err != nil {
		return "", err
	}
	return uri, nil
}

// UploadGCSDeployArtifact uploads the provided content as a deploy artifact. The objectSuffix must be provided
// to determine the URI to use for the GCS object, the URI is returned.
func UploadGCSDeployArtifact(ctx context.Context, gcsClient *storage.Client, deployRequest *DeployRequest, objectSuffix string, content *GCSUploadContent) (string, error) {
	if len(objectSuffix) == 0 {
		return "", fmt.Errorf("objectSuffix must be provided to upload a deploy artifact")
	}
	// For deploy the output gcs path is the path to a GCS directory.
	uri := fmt.Sprintf("%s/%s", deployRequest.OutputGCSPath, objectSuffix)
	if err := uploadGCS(ctx, gcsClient, uri, content); err != nil {
		return "", err
	}
	return uri, nil
}

// UploadGCSDeployResult uploads the provided deploy result to the GCS path expected by Cloud Deploy. Returns the GCS URI
// of the uploaded result.
func UploadGCSDeployResult(ctx context.Context, gcsClient *storage.Client, deployRequest *DeployRequest, deployResult *DeployResult) (string, error) {
	uri := fmt.Sprintf("%s/%s", deployRequest.OutputGCSPath, resultObjectSuffix)
	res, err := json.Marshal(deployResult)
	if err != nil {
		return "", fmt.Errorf("error marshalling deploy result: %v", err)
	}
	if err := uploadGCS(ctx, gcsClient, uri, &GCSUploadContent{Data: res}); err != nil {
		return "", err
	}
	return uri, nil
}

// downloadGCS downloads the GCS object for the specified URI to the provided local path.
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

// uploadGCS uploads the provided content to the specified URI.
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

// gcsObjectURI is used to split the object URI into the bucket and name.
type gcsObjectURI struct {
	// bucket the GCS object is in.
	bucket string
	// name of the GCS object.
	name string
}

// parseGCSURI parses the URI and returns the corresponding gcsObjectURI.
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
