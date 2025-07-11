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
	"os"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/cloud-deploy-samples/custom-targets/util/clouddeploy"
	"github.com/GoogleCloudPlatform/cloud-deploy-samples/packages/gcs"
	"github.com/mholt/archiver/v3"
)

// deployer implements the requestHandler interface for deploy requests.
type deployer struct {
	req       *clouddeploy.DeployRequest
	params    *params
	gcsClient *storage.Client
}

// process processes a deploy request and uploads succeeded or failed results to GCS for Cloud Deploy.
func (d *deployer) process(ctx context.Context) error {
	fmt.Println("Processing deploy request")

	res, err := d.deploy(ctx)
	if err != nil {
		fmt.Printf("Deploy failed: %v\n", err)
		dr := &clouddeploy.DeployResult{
			ResultStatus:   clouddeploy.DeployFailed,
			FailureMessage: err.Error(),
			Metadata: map[string]string{
				clouddeploy.CustomTargetSourceMetadataKey:    helmDeployerSampleName,
				clouddeploy.CustomTargetSourceSHAMetadataKey: clouddeploy.GitCommit,
			},
		}
		fmt.Println("Uploading failed deploy results")
		rURI, err := d.req.UploadResult(ctx, d.gcsClient, dr)
		if err != nil {
			return fmt.Errorf("error uploading failed deploy results: %v", err)
		}
		fmt.Printf("Uploaded failed deploy results to %s\n", rURI)
		return err
	}

	fmt.Println("Uploading deploy results")
	rURI, err := d.req.UploadResult(ctx, d.gcsClient, res)
	if err != nil {
		return fmt.Errorf("error uploading deploy results: %v", err)
	}
	fmt.Printf("Uploaded deploy results to %s\n", rURI)
	return nil
}

// deploy performs the following steps:
//  1. Run helm upgrade for the provided helm chart
//  2. Get the helm release manifest and upload to GCS as a deploy artifact.
//
// Returns either the deploy results or an error if the deploy failed.
func (d *deployer) deploy(ctx context.Context) (*clouddeploy.DeployResult, error) {
	fmt.Printf("Downloading helm configuration archive to %s\n", srcArchivePath)
	inURI, err := d.req.DownloadInput(ctx, d.gcsClient, renderedArchiveName, srcArchivePath)
	if err != nil {
		return nil, fmt.Errorf("unable to download deploy input with object suffix %s: %v", renderedArchiveName, err)
	}
	fmt.Printf("Downloaded helm configuration archive from %s\n", inURI)

	archiveFile, err := os.Open(srcArchivePath)
	if err != nil {
		return nil, fmt.Errorf("unable to open archive file %s: %v", srcArchivePath, err)
	}
	fmt.Printf("Unarchiving helm configuration in %s to %s\n", srcArchivePath, srcPath)
	if err := archiver.NewTarGz().Unarchive(archiveFile.Name(), srcPath); err != nil {
		return nil, fmt.Errorf("unable to unarchive helm configuration: %v", err)
	}

	fmt.Printf("Setting up cluster credentials for %s\n", d.params.gkeCluster)
	if _, err := gcloudClusterCredentials(d.params.gkeCluster); err != nil {
		return nil, fmt.Errorf("unable to set up cluster credentials: %v", err)
	}
	fmt.Printf("Finished setting up cluster credentials for %s\n", d.params.gkeCluster)

	// Use the pipeline ID as the helm release since this should be consistent.
	helmRelease := d.req.Pipeline
	chartPath := determineChartPath(d.params)
	hOpts := helmOptions{namespace: d.params.namespace}
	if _, err := helmUpgrade(helmRelease, chartPath, &helmUpgradeOptions{helmOptions: hOpts, timeout: d.params.upgradeTimeout}); err != nil {
		return nil, fmt.Errorf("error running helm upgrade: %v", err)
	}

	// After `helm upgrade` succeeds get the manifest to upload as the deploy artifact.
	manifest, err := helmGetManifest(helmRelease, &helmOptions{namespace: d.params.namespace})
	if err != nil {
		return nil, fmt.Errorf("error running helm get manifest aft upgrade: %v", err)
	}
	fmt.Println("Uploading helm release manifest as a deploy artifact")
	mURI, err := d.req.UploadArtifact(ctx, d.gcsClient, "manifest.yaml", &gcs.UploadContent{Data: manifest})
	if err != nil {
		return nil, fmt.Errorf("error uploading helm release manifest deploy artifact: %v", err)
	}

	dr := &clouddeploy.DeployResult{
		ResultStatus:  clouddeploy.DeploySucceeded,
		ArtifactFiles: []string{mURI},
		Metadata: map[string]string{
			clouddeploy.CustomTargetSourceMetadataKey:    helmDeployerSampleName,
			clouddeploy.CustomTargetSourceSHAMetadataKey: clouddeploy.GitCommit,
		},
	}
	return dr, nil
}
