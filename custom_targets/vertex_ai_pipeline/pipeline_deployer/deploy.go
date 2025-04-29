// deploy.go contains logic to deploy a pipeline to vertex AI.
package main

import (
	"context"
	"fmt"

	"google3/third_party/cloud_deploy_samples/custom_targets/util/clouddeploy/clouddeploy"
	"google3/third_party/golang/cloud_google_com/go/storage/v/v1/storage"
	"google3/third_party/golang/google_api/aiplatform/v1/aiplatform"
	"google3/third_party/golang/kubeyaml/yaml"
)

const aiDeployerSampleName = "clouddeploy-vertex-ai-pipeline-sample"

const localManifest = "manifest.yaml"

// deployer implements the handler interface to deploy a pipeline using the vertex AI API.
type deployer struct {
	gcsClient         *storage.Client
	aiPlatformService *aiplatform.Service
	params            *params
	req               *clouddeploy.DeployRequest
}

// process processes the Deploy request, and performs the vertex AI pipeline deployment.
func (d *deployer) process(ctx context.Context) error {
	fmt.Println("Processing deploy request")

	res, err := d.deploy(ctx)
	if err != nil {
		fmt.Printf("Deploy failed: %v\n", err)
		dr := &clouddeploy.DeployResult{
			ResultStatus:   clouddeploy.DeployFailed,
			FailureMessage: err.Error(),
		}
		d.addCommonMetadata(dr)
		fmt.Println("Uploading failed deploy results")
		rURI, err := d.req.UploadResult(ctx, d.gcsClient, dr)
		if err != nil {
			return fmt.Errorf("error uploading failed deploy results: %v", err)
		}
		fmt.Printf("Uploaded failed deploy results to %s\n", rURI)
		return err
	}
	d.addCommonMetadata(res)

	fmt.Println("Uploading successful deploy results")
	rURI, err := d.req.UploadResult(ctx, d.gcsClient, res)
	if err != nil {
		return fmt.Errorf("error uploading deploy results: %v", err)
	}
	fmt.Printf("Uploaded deploy results to %s\n", rURI)
	return nil
}

// deploy performs the Vertex AI pipeline deployment
func (d *deployer) deploy(ctx context.Context) (*clouddeploy.DeployResult, error) {

	if err := d.downloadManifest(ctx); err != nil {
		return nil, err
	}

	manifestData, err := d.applyPipeline(ctx, localManifest)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy pipeline: %v", err)
	}

	mURI, err := d.req.UploadArtifact(ctx, d.gcsClient, "manifest.yaml", &clouddeploy.GCSUploadContent{Data: manifestData})
	if err != nil {
		return nil, fmt.Errorf("error uploading deploy artifact: %v", err)
	}

	return &clouddeploy.DeployResult{
		ResultStatus:  clouddeploy.DeploySucceeded,
		ArtifactFiles: []string{mURI},
	}, nil
}

// downloadManifest downloads the rendered manifest from Google Cloud Storage to the local manifest file path
func (d *deployer) downloadManifest(ctx context.Context) error {
	fmt.Printf("Downloading deploy input manifest from %q.\n", d.req.ManifestGCSPath)

	downloadPath, err := d.req.DownloadManifest(ctx, d.gcsClient, localManifest)
	if err != nil {
		fmt.Printf("Unable to download deployed manifest from: %s.\n", d.req.ManifestGCSPath)
		return fmt.Errorf("unable to download deploy input from %s: %v", d.req.ManifestGCSPath, err)
	}

	fmt.Printf("Downloaded deploy input manifest from: %s\n", downloadPath)
	return nil
}

// addCommonMetadata inserts metadata into the deploy result that should be present
// regardless of deploy success or failure.
func (d *deployer) addCommonMetadata(rs *clouddeploy.DeployResult) {
	if rs.Metadata == nil {
		rs.Metadata = map[string]string{}
	}
	rs.Metadata[clouddeploy.CustomTargetSourceMetadataKey] = aiDeployerSampleName
	rs.Metadata[clouddeploy.CustomTargetSourceSHAMetadataKey] = clouddeploy.GitCommit
}

// applyModel deploys the CreatePipelineJobRequest parsed from `localManifest`
// it returns the CreatePipelineJobRequest object that was used in yaml format.
func (d *deployer) applyPipeline(ctx context.Context, localManifest string) ([]byte, error) {

	pipelineRequest, err := pipelineRequestFromManifest(localManifest)
	if err != nil {
		return nil, fmt.Errorf("unable to load CreatePipelineJobRequest from manifest: %v", err)
	}

	parent := fmt.Sprintf("projects/%s/locations/%s", d.params.project, d.params.location)

	if err := deployPipeline(ctx, d.aiPlatformService, parent, pipelineRequest); err != nil {
		return nil, fmt.Errorf("unable to deploy pipeline: %v", err)
	}
	return yaml.Marshal(pipelineRequest)
}
