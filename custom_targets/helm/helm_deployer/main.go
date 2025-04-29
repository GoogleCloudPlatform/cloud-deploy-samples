package main

import (
	"context"
	"fmt"
	"os"

	"google3/third_party/cloud_deploy_samples/custom_targets/util/clouddeploy/clouddeploy"
	"google3/third_party/golang/cloud_google_com/go/storage/v/v1/storage"
)

const (
	// The name of the Helm deployer sample, this is passed back to Cloud Deploy
	// as metadata in the render and deploy results.
	helmDeployerSampleName = "clouddeploy-helm-sample"
)

func main() {
	if err := do(); err != nil {
		fmt.Printf("err: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Done!")
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
	switch r := cloudDeployRequest.(type) {
	case *clouddeploy.RenderRequest:
		return &renderer{
			req:       r,
			params:    params,
			gcsClient: gcsClient,
		}, nil

	case *clouddeploy.DeployRequest:
		return &deployer{
			req:       r,
			params:    params,
			gcsClient: gcsClient,
		}, nil

	default:
		return nil, fmt.Errorf("received unsupported cloud deploy request type: %q", os.Getenv(clouddeploy.RequestTypeEnvKey))
	}
}
