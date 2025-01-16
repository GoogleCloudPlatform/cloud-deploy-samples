package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"cloud.google.com/go/storage"
)

// postdeployHookResult represents the json data in the results file for a
// postdeploy hook operation.
type postdeployHookResult struct {
	Metadata map[string]string `json:"metadata,omitempty"`
}

// uploadResult uploads the provided deploy result to the Cloud Storage path where Cloud Deploy expects it.
func uploadResult(ctx context.Context, gcsClient *storage.Client, deployHookResult *postdeployHookResult) error {
	// This environment variable is provided by Cloud Deploy and the value is
	// where to upload a results file.
	uri := os.Getenv("CLOUD_DEPLOY_OUTPUT_GCS_PATH")
	jsonResult, err := json.Marshal(deployHookResult)
	if err != nil {
		return fmt.Errorf("error marshalling postdeploy hook result: %v", err)
	}
	if err := uploadGCS(ctx, gcsClient, uri, jsonResult); err != nil {
		return err
	}
	return nil
}

// uploadGCS uploads the provided content to the specified Cloud Storage URI.
func uploadGCS(ctx context.Context, gcsClient *storage.Client, gcsURI string, content []byte) error {

	gcsObjURI, err := parseGCSURI(gcsURI)
	if err != nil {
		return err
	}
	w := gcsClient.Bucket(gcsObjURI.bucket).Object(gcsObjURI.name).NewWriter(ctx)
	if _, err := w.Write(content); err != nil {
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
