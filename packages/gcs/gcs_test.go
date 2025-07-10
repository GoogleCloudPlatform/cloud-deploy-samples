package gcs

import (
	"bytes"
	"context"
	"os"
	"path"
	"path/filepath"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/fsouza/fake-gcs-server/fakestorage"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/mholt/archiver/v3"
	"google3/testing/gobase/googletest"
)

func TestParseGCSURI(t *testing.T) {
	testCases := []struct {
		desc       string
		uri        string
		wantGcsObj gcsObjectURI
	}{
		{
			desc: "success",
			uri:  "gs://fake_bucket//fake_object",
			wantGcsObj: gcsObjectURI{
				bucket: "fake_bucket",
				name:   "fake_object",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			gotGcsObj, err := parseGCSURI(tc.uri)
			if err != nil {
				t.Fatalf("Failed to parse URI err is %v", err)
			}
			if diff := cmp.Diff(gotGcsObj, tc.wantGcsObj, cmpopts.EquateComparable(gcsObjectURI{})); diff != "" {
				t.Errorf("parseGCSURI(%v) got: '%+v', want '%+v'",
					tc.uri,
					gotGcsObj,
					tc.wantGcsObj)
			}
		})
	}
}

func TestParseGCSURIInvalid(t *testing.T) {
	testCases := []struct {
		desc string
		uri  string
	}{
		{
			desc: "wrong schema",
			uri:  "http://fake_bucket//fake_object",
		},
		{
			desc: "empty uri",
			uri:  "",
		},
		{
			desc: "empty Bucket",
			uri:  "gs:////fake_object",
		},
		{
			desc: "empty object",
			uri:  "gs://fake_bucket//",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			_, errMsg := parseGCSURI(tc.uri)
			if errMsg == nil {
				t.Fatalf("parseGCSURI(%v) succeeded for invalid URI, want error", tc.uri)
			}
		})
	}
}

func TestDownloadGCS(t *testing.T) {
	ctx := context.Background()
	fakeContent := "hello world"
	gcsClient := CreateGCSClient(t, []byte(fakeContent), bucket, "test.yaml")

	tests := []struct {
		desc        string
		gcsURI      string
		localPath   string
		wantContent string
		wantFiles   []string
	}{
		{
			desc:        "success",
			gcsURI:      gsPrefix + bucket + "/" + "test.yaml",
			localPath:   filepath.Join(t.TempDir(), "workspace"),
			wantContent: fakeContent,
			wantFiles:   files,
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			_, err := Download(ctx, gcsClient, test.gcsURI, test.localPath)
			if err != nil {
				t.Fatalf("Failed to download GCS bucket content locally err is %v, gcsUri: %q, client: %v", err, test.gcsURI, gcsClient)
			}
			// Read the contents of the local file.
			got, err := os.ReadFile(test.localPath)
			if err != nil {
				t.Fatalf("Couldn't read file content, err: %v", err)
			}
			if bytes.Compare([]byte(test.wantContent), got) != 0 {
				t.Fatalf("Expected file to contain %q, got: %q",
					test.wantContent,
					string(got))
			}
		})
	}

	invalidtc := []struct {
		desc      string
		gcsURI    string
		localPath string
	}{
		{
			desc:      "non-existent GCS URI",
			gcsURI:    "gs://non-existent-bucket/nonexistent",
			localPath: filepath.Join(t.TempDir(), "workspace"),
		},
		{
			desc:      "non-existent GCS object",
			gcsURI:    "gs://example-bucket/nonexistent",
			localPath: filepath.Join(t.TempDir(), "workspace"),
		},
		{
			desc:      "empty URI",
			gcsURI:    "",
			localPath: filepath.Join(googletest.TestTmpDir, "workspace"),
		},
	}
	for _, test := range invalidtc {
		t.Run(test.desc, func(t *testing.T) {
			_, err := Download(ctx, gcsClient, test.gcsURI, test.localPath)
			if err == nil {
				t.Errorf("downloadGCS(%q, %q) succeeded for non-existent URI, want error",
					test.gcsURI,
					test.localPath)
			}
		})
	}
}

func TestUploadGCS(t *testing.T) {
	ctx := context.Background()
	fakeContent := "hello world"
	fakeObj := "obj"
	fakeGCS := fakestorage.NewServer([]fakestorage.Object{})
	fakeGCS.CreateBucketWithOpts(fakestorage.CreateBucketOpts{Name: bucket})

	fakeContentForFile := "hellooo"
	fullFileName := filepath.Join(t.TempDir(), "testing.yaml")
	if err := os.WriteFile(fullFileName, []byte(fakeContentForFile), os.ModePerm); err != nil {
		t.Fatalf("couldn't write test file %v err is %v", fullFileName, err)
	}

	tests := []struct {
		desc        string
		gcsURI      string
		content     *UploadContent
		wantContent string
	}{
		{
			desc:        "success when uploading a byte array",
			gcsURI:      gsPrefix + bucket + "/" + fakeObj,
			content:     &UploadContent{Data: []byte(fakeContent)},
			wantContent: fakeContent,
		},
		{
			desc:        "success when uploading content from a file",
			gcsURI:      gsPrefix + bucket + "/" + fakeObj,
			content:     &UploadContent{LocalPath: fullFileName},
			wantContent: fakeContentForFile,
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := Upload(ctx, fakeGCS.Client(), test.gcsURI, test.content)
			if err != nil {
				t.Fatalf("Failed to upload GCS bucket content, err is %v", err)
			}
			o, err := fakeGCS.GetObject(bucket, fakeObj)
			if err != nil {
				t.Fatalf("Failed to get GCS object %v, err: %v", fakeObj, err)
			}
			if string(o.Content) != test.wantContent {
				t.Errorf("uploadGCS failed with wrong content: got %q, want %q", o.Content, test.wantContent)
			}

			fakeGCS.Stop()
		})
	}

	invalidtc := []struct {
		desc    string
		gcsURI  string
		content *UploadContent
	}{
		{
			desc:    "non-existent GCS URI",
			gcsURI:  "gs://non-existent-bucket/nonexistent",
			content: &UploadContent{Data: []byte(fakeContent)},
		},
		{
			desc:    "empty URI",
			gcsURI:  "",
			content: &UploadContent{Data: []byte(fakeContent)},
		},
	}
	for _, test := range invalidtc {
		t.Run(test.desc, func(t *testing.T) {
			err := Upload(ctx, fakeGCS.Client(), test.gcsURI, test.content)
			if err == nil {
				t.Errorf("uploadGCS(%q, %q) succeeded, want error",
					test.gcsURI,
					test.content)
			}
		})
	}
}

const (
	gsPrefix = "gs://"
	tarDest  = "input.tar.gz"
	bucket   = "example-bucket"
)

var files = []string{"test.yaml", "source_code.go"}

// CreateGCSClient creates a fake server and populates it with the given bucket, object and content.
func CreateGCSClient(t *testing.T, content []byte, bucketName, objName string) *storage.Client {
	t.Helper()

	server := fakestorage.NewServer([]fakestorage.Object{
		fakestorage.Object{
			Content: content,
			ObjectAttrs: fakestorage.ObjectAttrs{
				BucketName: bucketName,
				Name:       objName,
			},
		},
	})
	t.Cleanup(server.Stop)
	return server.Client()
}

// CreateTarFile creates a tar file that contains the given files. The tarDest
// should end in a tar file extension, e.g. "foo.tar.gz".
func CreateTarFile(t *testing.T, tarDest string, files []string) []byte {
	t.Helper()

	var sources []string
	for _, f := range files {
		fullFileName := filepath.Join(t.TempDir(), f)
		if err := os.WriteFile(fullFileName, []byte(""), os.ModePerm); err != nil {
			t.Fatalf("couldn't write test file %v err is %v", fullFileName, err)
		}
		sources = append(sources, fullFileName)
	}

	// Create the fake tar dest file and archive the source files into it.
	fullDestName := path.Join(t.TempDir(), tarDest)
	if err := archiver.Archive(sources, fullDestName); err != nil {
		t.Fatalf("failed to create archive: %v", err)
	}
	data, err := os.ReadFile(fullDestName)
	if err != nil {
		t.Fatalf("failed to read archive file into memory: %v", err)
	}
	return data
}
