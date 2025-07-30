// Copyright 2023 Google LLC

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     https://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	helmBin   = "helm"
	gcloudBin = "gcloud"
)

// helmOptions configures the args provided to `helm`.
type helmOptions struct {
	namespace string
}

// helmTemplateOptions configures the args provided to `helm template`.
type helmTemplateOptions struct {
	helmOptions
	lookup   bool
	validate bool
}

// helmTemplate runs `helm template` for the provided release name and chart path with the
// provided options. The output from this command is not written to stdout. Returns the
// manifest in YAML format.
func helmTemplate(releaseName, chartPath string, opts *helmTemplateOptions) ([]byte, error) {
	args := []string{"template", releaseName, chartPath, "--include-crds"}
	if opts.lookup {
		args = append(args, "--dry-run=server")
	}
	if opts.validate {
		args = append(args, "--validate")
	}
	if len(opts.helmOptions.namespace) > 0 {
		args = append(args, fmt.Sprintf("--namespace=%s", opts.helmOptions.namespace))
	}
	return runCmd(helmBin, args, true)
}

// helmUpgradeOptions configures the args provided to `helm upgrade`.
type helmUpgradeOptions struct {
	helmOptions
	timeout string
}

// helmUpgrade runs `helm upgrade` for the provided release and chart path with the
// provided options.
func helmUpgrade(releaseName, chartPath string, opts *helmUpgradeOptions) ([]byte, error) {
	args := []string{"upgrade", releaseName, chartPath, "--install", "--wait", "--wait-for-jobs"}
	if len(opts.timeout) != 0 {
		args = append(args, fmt.Sprintf("--timeout=%s", opts.timeout))
	}
	if len(opts.helmOptions.namespace) > 0 {
		args = append(args, fmt.Sprintf("--namespace=%s", opts.helmOptions.namespace))
		args = append(args, "--create-namespace")
	}
	return runCmd(helmBin, args, false)
}

// helmGetManifest runs `helm get manifest` for the provided release name. The output
// from this command is not written to stdout.
func helmGetManifest(releaseName string, opts *helmOptions) ([]byte, error) {
	args := []string{"get", "manifest", releaseName}
	if len(opts.namespace) > 0 {
		args = append(args, fmt.Sprintf("--namespace=%s", opts.namespace))
	}
	return runCmd(helmBin, args, true)
}

// gkeClusterRegex represents the regex that a GKE cluster resource name needs to match.
var gkeClusterRegex = regexp.MustCompile("^projects/([^/]+)/locations/([^/]+)/clusters/([^/]+)$")

// gcloudClusterCredentials runs `gcloud container clusters get-credentials` to set up
// the cluster credentials.
func gcloudClusterCredentials(gkeCluster string) ([]byte, error) {
	m := gkeClusterRegex.FindStringSubmatch(gkeCluster)
	if len(m) == 0 {
		return nil, fmt.Errorf("invalid GKE cluster name: %s", gkeCluster)
	}
	args := []string{"container", "clusters", "get-credentials", m[3], fmt.Sprintf("--region=%s", m[2]), fmt.Sprintf("--project=%s", m[1])}
	return runCmd(gcloudBin, args, false)
}

// runCmd starts and waits for the provided command with args to complete. If the command
// succeeds it returns the stdout of the command.
func runCmd(binPath string, args []string, closeOSStdout bool) ([]byte, error) {
	fmt.Printf("Running the following command: %s %s\n", binPath, args)
	cmd := exec.Command(binPath, args...)

	var stderr bytes.Buffer
	errWriter := io.MultiWriter(&stderr, os.Stderr)
	cmd.Stderr = errWriter

	var stdout bytes.Buffer
	if closeOSStdout {
		cmd.Stdout = &stdout
	} else {
		cmd.Stdout = io.MultiWriter(&stdout, os.Stdout)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("error running command: %v\n%s", err, stderr.Bytes())
	}
	return stdout.Bytes(), nil
}
