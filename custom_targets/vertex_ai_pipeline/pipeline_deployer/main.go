package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"google3/third_party/cloud_deploy_samples/custom_targets/util/clouddeploy/clouddeploy"
	"google3/third_party/golang/cloud_google_com/go/storage/v/v1/storage"
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
		return fmt.Errorf("unable to create gcs client: %v", err)
	}

	flag.Parse()

	req, err := clouddeploy.DetermineRequest(ctx, gcsClient, []string{"CANARY"})
	if err != nil {
		return err
	}

	params, err := determineParams()
	if err != nil {
		return fmt.Errorf("unable to parse params: %v", err)
	}

	aiPlatformService, err := newAIPlatformService(ctx, params.location)
	if err != nil {
		return fmt.Errorf("unable to create aiplatform.Service object : %v", err)
	}

	handler, err := createRequestHandler(req, params, gcsClient, aiPlatformService)
	if err != nil {
		return fmt.Errorf("unable to create request handler: %v", err)
	}

	return handler.process(ctx)
}
