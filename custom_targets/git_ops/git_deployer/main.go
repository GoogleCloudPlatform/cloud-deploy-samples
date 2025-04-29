package main

import (
	"context"
	"fmt"
	"os"

	"google3/third_party/cloud_deploy_samples/custom_targets/util/clouddeploy/clouddeploy"
	secretmanager "google3/third_party/golang/cloud_google_com/go/secretmanager/v/v0/apiv1/secretmanager"
	"google3/third_party/golang/cloud_google_com/go/storage/v/v1/storage"
)

const (
	// The name of the Git deployer sample, this is passed back to Cloud Deploy
	// as metadata in the deploy results.
	gitDeployerSampleName = "clouddeploy-git-ops-sample"
)

func main() {
	if err := do(); err != nil {
		fmt.Printf("err: %v\n", err)
		os.Exit(1)
	}

}

func do() error {
	ctx := context.Background()
	gcsClient, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("unable to create cloud storage client: %v", err)
	}
	req, err := clouddeploy.DetermineRequest(ctx, gcsClient, []string{})
	if err != nil {
		return fmt.Errorf("unable to determine cloud deploy request: %v", err)
	}
	params, err := determineParams()
	if err != nil {
		return fmt.Errorf("unable to determine params: %v", err)
	}
	h, err := createRequestHandler(ctx, req, params, gcsClient)
	if err != nil {
		return err
	}
	return h.process(ctx)
}

// requestHandler interface provides methods for handling the Cloud Deploy request.
type requestHandler interface {
	// Process processes the Cloud Deploy request.
	process(ctx context.Context) error
}

// createRequestHandler creates a requestHandler for the provided Cloud Deploy request.
func createRequestHandler(ctx context.Context, cloudDeployRequest interface{}, params *params, gcsClient *storage.Client) (requestHandler, error) {
	// The git deployer only supports deploy. If a render request is received then a not supported result will be
	// uploaded to Cloud Storage in order to provide Cloud Deploy with context on why the render failed.
	switch r := cloudDeployRequest.(type) {
	case *clouddeploy.RenderRequest:
		fmt.Println("Received render request from Cloud Deploy, which is not supported. Uploading not supported render results")
		res := &clouddeploy.RenderResult{
			ResultStatus:   clouddeploy.RenderNotSupported,
			FailureMessage: fmt.Sprintf("Render is not supported by %s", gitDeployerSampleName),
			Metadata: map[string]string{
				clouddeploy.CustomTargetSourceMetadataKey:    gitDeployerSampleName,
				clouddeploy.CustomTargetSourceSHAMetadataKey: clouddeploy.GitCommit,
			},
		}
		rURI, err := r.UploadResult(ctx, gcsClient, res)
		if err != nil {
			return nil, fmt.Errorf("error uploading not supported render results: %v", err)
		}
		fmt.Printf("Uploaded not supported render results to %s\n", rURI)
		return nil, fmt.Errorf("render not supported by %s", gitDeployerSampleName)

	case *clouddeploy.DeployRequest:
		smClient, err := secretmanager.NewClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("unable to create secret manager client: %v", err)
		}

		return &deployer{
			req:       r,
			params:    params,
			gcsClient: gcsClient,
			smClient:  smClient,
		}, nil

	default:
		return nil, fmt.Errorf("received unsupported cloud deploy request type: %q", os.Getenv(clouddeploy.RequestTypeEnvKey))
	}
}
