// Package gcs provides functions for interacting with Google Cloud Storage.
package gcs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"cloud.google.com/go/storage"
)

// ResultObjectSuffix is the Cloud Storage object suffix for the expected results file.
const ResultObjectSuffix = "results.json"

// Download downloads the Cloud Storage object for the specified URI to the provided local path.
func Download(ctx context.Context, gcsClient *storage.Client, gcsURI, localPath string) (*os.File, error) {
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

	if _, err := io.Copy(file, r); err != nil {
		return nil, err
	}
	return file, nil
}

// UploadContent is used as a parameter for the various GCS upload functions that points
// to the source of the content to upload.
type UploadContent struct {
	// Content is this byte array.
	Data []byte
	// Content is in the file at this local path.
	LocalPath string
}

// Upload uploads the provided content to the specified Cloud Storage URI.
func Upload(ctx context.Context, gcsClient *storage.Client, gcsURI string, content *UploadContent) error {
	// Determine the source of the content to upload.
	var contentData []byte
	switch {
	case len(content.Data) != 0 && len(content.LocalPath) != 0:
		return fmt.Errorf("unable to determine the content to upload to GCS, both data and a local path were provided")
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
