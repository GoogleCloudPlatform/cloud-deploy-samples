// Package gcs provides utilities for uploading analysis results to GCS.
package gcs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"cloud.google.com/go/storage"
	cdenv "github.com/GoogleCloudPlatform/cloud-deploy-samples/packages/cdenv"
	"github.com/GoogleCloudPlatform/cloud-deploy-samples/packages/gcs"
)

// AnalysisMetadata represents the Analysis result metadata that will be uploaded to GCS.
type AnalysisMetadata struct {
	// Metadata contains metadata associated with the analysis result.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// UploadResult uploads the result to GCS.
func UploadResult(ctx context.Context, analysisMetadata *AnalysisMetadata, client *storage.Client) error {
	data, err := json.Marshal(analysisMetadata)
	if err != nil {
		return fmt.Errorf("failed to marshal analysis metadata: %v, error: %w", analysisMetadata, err)
	}
	// Get the GCS URI where the results file should be uploaded. The full path is in the format of
	// {outputPath}/{gcs.ResultObjectSuffix}.
	outputPath := os.Getenv(cdenv.OutputGCSEnvKey)
	uri := fmt.Sprintf("%s/%s", outputPath, gcs.ResultObjectSuffix)
	return gcs.Upload(ctx, client, uri, &gcs.UploadContent{Data: data})
}
