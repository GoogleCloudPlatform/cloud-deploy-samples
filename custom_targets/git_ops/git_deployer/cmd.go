package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
)

const (
	kubectlBin = "kubectl"
	gcloudBin  = "gcloud"
)

// gkeClusterRegex represents the regex that a GKE cluster resource name needs to match.
var gkeClusterRegex = regexp.MustCompile("^projects/([^/]+)/locations/([^/]+)/clusters/([^/]+)$")

// gcloudClusterCredentials runs `gcloud container clusters get-crendetials` to set up
// the cluster credentials.
func gcloudClusterCredentials(gkeCluster string) ([]byte, error) {
	m := gkeClusterRegex.FindStringSubmatch(gkeCluster)
	if len(m) == 0 {
		return nil, fmt.Errorf("invalid GKE cluster name: %s", gkeCluster)
	}
	args := []string{"container", "clusters", "get-credentials", m[3], fmt.Sprintf("--region=%s", m[2]), fmt.Sprintf("--project=%s", m[1])}
	return runCmd(gcloudBin, args, "", true)
}

// verifyResourceExists gets the Kubernetes resource if it exists.
func verifyResourceExists(rt, rn, ns string) ([]byte, error) {
	args := []string{"get", rt, rn, fmt.Sprintf("-n=%s", ns)}
	return runCmd(kubectlBin, args, "", true)
}

// queryPath queries the JSON path of a Kubernetes resource.
func queryPath(rt, rn, ns, path string) ([]byte, error) {
	args := []string{"get", rt, rn, fmt.Sprintf("-n=%s", ns), fmt.Sprintf("-o=jsonpath=%s", path)}
	return runCmd(kubectlBin, args, "", true)
}

// runCmd starts and waits for the provided command with args to complete. If the command
// succeeds it returns the stdout of the command.
func runCmd(binPath string, args []string, dir string, logCmd bool) ([]byte, error) {
	if logCmd {
		fmt.Printf("Running the following command: %s %s\n", binPath, args)
	}
	cmd := exec.Command(binPath, args...)
	cmd.Dir = dir

	var stderr bytes.Buffer
	errWriter := io.MultiWriter(&stderr, os.Stderr)
	cmd.Stderr = errWriter

	var stdout bytes.Buffer
	outWriter := io.MultiWriter(&stdout, os.Stdout)
	cmd.Stdout = outWriter

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("error running command: %v\n%s", err, stderr.Bytes())
	}
	return stdout.Bytes(), nil
}
