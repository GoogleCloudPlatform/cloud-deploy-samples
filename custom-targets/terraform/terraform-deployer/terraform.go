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
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
)

const (
	terraformBin = "terraform"
)

// terraformInitOptions configures the args provided to `terraform init`.
type terraformInitOptions struct {
	disableBackendInitialization bool
	disableModuleDownloads       bool
}

// terraformInit runs `terraform init` in the provided directory.
func terraformInit(workingDir string, opts *terraformInitOptions) ([]byte, error) {
	args := []string{"init", "-no-color"}
	if opts.disableBackendInitialization {
		args = append(args, "-backend=false")
	}
	if opts.disableModuleDownloads {
		args = append(args, "-get=false")
	}
	fmt.Printf("Running terraform init in %s\n", workingDir)
	return runCmd(terraformBin, args, false, setWorkingDir(workingDir))
}

// terraformValidate runs `terraform validate` in the provided directory.
func terraformValidate(workingDir string) ([]byte, error) {
	args := []string{"validate", "-no-color"}
	fmt.Printf("Running terraform validate in %s\n", workingDir)
	return runCmd(terraformBin, args, false, setWorkingDir(workingDir))
}

// terraformPlan runs `terraform plan` in the provided directory and creates the
// plan in the working directory with the provided file name.
func terraformPlan(workingDir, planFile string) ([]byte, error) {
	args := []string{"plan", "-no-color", fmt.Sprintf("-out=%s", planFile)}
	fmt.Printf("Running terraform plan in %s\n", workingDir)
	return runCmd(terraformBin, args, false, setWorkingDir(workingDir))
}

// terraformShowPlan runs `terraform show` in the provided directory for a provided
// plan file. The output from this command is not written to stdout.
func terraformShowPlan(workingDir, planFile string) ([]byte, error) {
	args := []string{"show", "-no-color", planFile}
	fmt.Printf("Running terraform show plan in %s\n", workingDir)
	return runCmd(terraformBin, args, true, setWorkingDir(workingDir))
}

// terraformApplyOptions configures the args provided to `terraform apply`.
type terraformApplyOptions struct {
	applyParallelism int
	lockTimeout      string
}

// terraformApply runs `terraform apply` in the provided directory.
func terraformApply(workingDir string, opts *terraformApplyOptions) ([]byte, error) {
	args := []string{"apply", "-auto-approve", "-no-color"}
	if len(opts.lockTimeout) != 0 {
		args = append(args, fmt.Sprintf("-lock-timeout=%s", opts.lockTimeout))
	}
	if opts.applyParallelism > 0 {
		args = append(args, fmt.Sprintf("-parallelism=%d", opts.applyParallelism))
	}
	fmt.Printf("Running terraform apply in %s\n", workingDir)
	return runCmd(terraformBin, args, false, setWorkingDir(workingDir))
}

// terraformShowState runs `terraform show` in the provided directory. The output
// from this command is not written to stdout.
func terraformShowState(workingDir string) ([]byte, error) {
	args := []string{"show", "-json"}
	fmt.Printf("Running terraform show in %s\n", workingDir)
	out, err := runCmd(terraformBin, args, true, setWorkingDir(workingDir))
	if err != nil {
		return nil, err
	}
	return addIndentationToJSON(out)
}

// addIndentationToJson returns a copy of the provided JSON with indentation added.
// This is used to make the data more human-readable.
func addIndentationToJSON(in []byte) ([]byte, error) {
	var pjson bytes.Buffer
	if err := json.Indent(&pjson, in, "", "    "); err != nil {
		return nil, fmt.Errorf("error adding indentation to json: %v", err)
	}
	return pjson.Bytes(), nil
}

// commandOption configures an exec.Cmd object with additional options.
type commandOption func(ce *exec.Cmd)

// setWorkingDir returns a commandOption for setting the working directory.
func setWorkingDir(workingDir string) commandOption {
	return func(cmd *exec.Cmd) {
		cmd.Dir = workingDir
	}
}

// runCmd starts and waits for the provided command with args to complete. If the command
// succeeds it returns the stdout of the command.
func runCmd(binPath string, args []string, closeOSStdout bool, options ...commandOption) ([]byte, error) {
	fmt.Printf("Running the following command: %s %s\n", binPath, args)
	cmd := exec.Command(binPath, args...)

	var stderr bytes.Buffer
	cmd.Stderr = io.MultiWriter(&stderr, os.Stderr)

	var stdout bytes.Buffer
	if closeOSStdout {
		cmd.Stdout = &stdout
	} else {
		cmd.Stdout = io.MultiWriter(&stdout, os.Stdout)
	}

	for _, opt := range options {
		opt(cmd)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("error running command: %v\n%s", err, stderr.Bytes())
	}
	return stdout.Bytes(), nil
}
