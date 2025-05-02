package main

import (
	"testing"
)

// Tests that pipelineRequestFromManifest fails when given an incorrect path. Does not test correct path or incomplete file!
func TestPipelineRequestFromManifest(t *testing.T) {
	_, err := pipelineRequestFromManifest("")
	if err == nil {
		t.Errorf("Expected: error, Actual: %s", err)
	}

	_, err = pipelineRequestFromManifest("testPath")
	if err == nil {
		t.Errorf("Expected: error, Actual: %s", err)
	}

	_, err = pipelineRequestFromManifest(" ")
	if err == nil {
		t.Errorf("Expected: error, Actual: %s", err)
	}
}
