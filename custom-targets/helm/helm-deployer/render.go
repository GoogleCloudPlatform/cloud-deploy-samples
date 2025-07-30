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
	"path"
	"time"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/cloud-deploy-samples/custom-targets/util/clouddeploy"
)

const (
	// Path to use when downloading the source input archive file.
	srcArchivePath = "/workspace/archive.tgz"
	// Path to use when unarchiving the source input.
	srcPath = "/workspace/source"
	// Name of the archive uploaded at render time that will be downloaded at deploy time.
	renderedArchiveName = "helm-archive.tgz"
)

var (
	// Default chart path used if not provided as a deploy parameter.
	defaultChartPath = path.Join(srcPath, "mychart")
)

// renderer implements the requestHandler interface for render requests.
type renderer struct {
	req       *clouddeploy.RenderRequest
	params    *params
	gcsClient *storage.Client
}

// process processes a render request and uploads succeeded or failed results to GCS for Cloud Deploy.
func (r *renderer) process(ctx context.Context) error {
	fmt.Println("Processing render request")

	res, err := r.render(ctx)
	if err != nil {
		fmt.Printf("Render failed: %v\n", err)
		rr := &clouddeploy.RenderResult{
			ResultStatus:   clouddeploy.RenderFailed,
			FailureMessage: err.Error(),
			Metadata: map[string]string{
				clouddeploy.CustomTargetSourceMetadataKey:    helmDeployerSampleName,
				clouddeploy.CustomTargetSourceSHAMetadataKey: clouddeploy.GitCommit,
			},
		}
		fmt.Println("Uploading failed render results")
		rURI, err := r.req.UploadResult(ctx, r.gcsClient, rr)
		if err != nil {
			return fmt.Errorf("error uploading failed render results: %v", err)
		}
		fmt.Printf("Uploaded failed render results to %s\n", rURI)
		return err
	}

	fmt.Println("Uploading render results")
	rURI, err := r.req.UploadResult(ctx, r.gcsClient, res)
	if err != nil {
		return fmt.Errorf("error uploading render results: %v", err)
	}
	fmt.Printf("Uploaded render results to %s\n", rURI)
	return nil
}

// render performs the following steps:
//  1. Run helm template for the provided helm chart to produce a manifest
//  2. Upload the manifest to GCS to use as the Cloud Deploy Release inspector artifact.
//  3. Upload the archived helm configuration to GCS so it can be used at deploy time.
//
// Returns either the render results or an error if the render failed.
func (r *renderer) render(ctx context.Context) (*clouddeploy.RenderResult, error) {
	fmt.Printf("Downloading render input archive to %s and unarchiving to %s\n", srcArchivePath, srcPath)
	inURI, err := r.req.DownloadAndUnarchiveInput(ctx, r.gcsClient, srcArchivePath, srcPath)
	if err != nil {
		return nil, fmt.Errorf("unable to download and unarchive render input: %v", err)
	}
	fmt.Printf("Downloaded render input archive from %s\n", inURI)

	// If template lookup or template validation is enabled then connect to the cluster at render time.
	if r.params.templateLookup || r.params.templateValidate {
		fmt.Printf("Helm template lookup or validate enabled. Setting up cluster credentials for %s\n", r.params.gkeCluster)
		if _, err := gcloudClusterCredentials(r.params.gkeCluster); err != nil {
			return nil, fmt.Errorf("unable to set up cluster credentials: %v", err)
		}
		fmt.Printf("Finished setting up cluster credentials for %s\n", r.params.gkeCluster)
	}

	// Use the pipeline ID as the helm release since this should be consistent.
	helmRelease := r.req.Pipeline
	chartPath := determineChartPath(r.params)
	hOpts := helmOptions{namespace: r.params.namespace}
	templateOut, err := helmTemplate(helmRelease, chartPath, &helmTemplateOptions{helmOptions: hOpts, lookup: r.params.templateLookup, validate: r.params.templateValidate})
	if err != nil {
		return nil, fmt.Errorf("error running helm template: %v", err)
	}

	tBytes, err := time.Now().MarshalText()
	if err != nil {
		return nil, fmt.Errorf("unable to marshal current time: %v", err)
	}
	// Add a comment at the top of the manifest indicating that it's not used at deploy time.
	manifest := []byte(fmt.Sprintf("# Manifest generated at %s by helm template.\n# This manifest is not used when performing the deploy, instead the same helm chart used to produce this manifest is provided to helm upgrade.\n", tBytes))
	manifest = append(manifest, templateOut...)

	fmt.Println("Uploading manifest from helm template")
	mURI, err := r.req.UploadArtifact(ctx, r.gcsClient, "manifest.yaml", &clouddeploy.GCSUploadContent{Data: manifest})
	if err != nil {
		return nil, fmt.Errorf("error uploading manifest: %v", err)
	}
	fmt.Printf("Uploaded manifest from helm template to %s\n", mURI)

	fmt.Println("Uploading archived helm configuration for use at deploy time")
	ahURI, err := r.req.UploadArtifact(ctx, r.gcsClient, renderedArchiveName, &clouddeploy.GCSUploadContent{LocalPath: srcArchivePath})
	if err != nil {
		return nil, fmt.Errorf("error uploading archived helm configuration: %v", err)
	}
	fmt.Printf("Uploaded archived helm configuration to %s\n", ahURI)

	rr := &clouddeploy.RenderResult{
		ResultStatus: clouddeploy.RenderSucceeded,
		ManifestFile: mURI,
		Metadata: map[string]string{
			clouddeploy.CustomTargetSourceMetadataKey:    helmDeployerSampleName,
			clouddeploy.CustomTargetSourceSHAMetadataKey: clouddeploy.GitCommit,
		},
	}
	return rr, nil
}

// determineChartPath determines the path to the helm chart based on the deploy parameters provided.
func determineChartPath(params *params) string {
	// If a path to the helm chart is provided then use it, otherwise default to "mychart" directory.
	chartPath := defaultChartPath
	if len(params.configPath) != 0 {
		chartPath = path.Join(srcPath, params.configPath)
	}
	return chartPath
}
