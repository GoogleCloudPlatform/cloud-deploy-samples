package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/cloud-deploy-samples/packages/cdenv"
	"github.com/GoogleCloudPlatform/cloud-deploy-samples/packages/gcs"
)

// postdeployHookResult represents the json data in the results file for a
// postdeploy hook operation.
type postdeployHookResult struct {
	Metadata map[string]string `json:"metadata,omitempty"`
}

// uploadResult uploads the provided deploy result to the Cloud Storage path where Cloud Deploy expects it.
func uploadResult(ctx context.Context, gcsClient *storage.Client, deployHookResult *postdeployHookResult) error {
	// Get the GCS URI where the results file should be uploaded.
	uri := os.Getenv(cdenv.OutputGCSEnvKey)
	jsonResult, err := json.Marshal(deployHookResult)
	if err != nil {
		return fmt.Errorf("error marshalling postdeploy hook result: %v", err)
	}
	if err := gcs.Upload(ctx, gcsClient, uri, &gcs.UploadContent{Data: jsonResult}); err != nil {
		return err
	}
	return nil
}
