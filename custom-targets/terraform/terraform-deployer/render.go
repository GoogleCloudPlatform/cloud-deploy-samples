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
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/cloud-deploy-samples/custom-targets/util/clouddeploy"
	"github.com/GoogleCloudPlatform/cloud-deploy-samples/packages/gcs"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/mholt/archiver/v3"
	"github.com/zclconf/go-cty/cty"
)

const (
	// Path to use when downloading the source input archive file.
	srcArchivePath = "/workspace/archive.tgz"
	// Path to use when unarchiving the source input.
	srcPath = "/workspace/source"
	// File name to use for the generated Terraform backend configuration.
	backendFileName = "backend.tf"
	// File name to use for the generated variables file.
	autoTFVarsFileName = "clouddeploy.auto.tfvars"
	// File name to use for the speculative Terraform plan.
	speculativePlanFileName = "clouddeploy-speculative-tfplan"
	// The directory within the Terraform configuration where providers are installed.
	providersDirName = ".terraform/providers"
	// Name of the release inspector artifact. This contains the contents of the generated variables file
	// and the speculative Terraform plan.
	inspectorArtifactName = "clouddeploy-release-inspector-artifact"
	// Name of the rendered archive. The rendered archive contains the Terraform configuration after
	// the rendering has completed.
	renderedArchiveName = "terraform-archive.tgz"
)

var (
	// Path to use when creating the release inspector artifact.
	inspectorArtifactPath = fmt.Sprintf("/workspace/%s", inspectorArtifactName)
)

// renderer implements the requestHandler interface for render requests.
type renderer struct {
	req       *clouddeploy.RenderRequest
	params    *params
	gcsClient *storage.Client
}

// process processes a render request and uploads succeeded or failed results to GCS for Cloud Deploy.
func (r *renderer) process(ctx context.Context) error {
	fmt.Println("Processing render request")

	res, err := r.render(ctx)
	if err != nil {
		fmt.Printf("Render failed: %v\n", err)
		rr := &clouddeploy.RenderResult{
			ResultStatus:   clouddeploy.RenderFailed,
			FailureMessage: err.Error(),
			Metadata: map[string]string{
				clouddeploy.CustomTargetSourceMetadataKey:    tfDeployerSampleName,
				clouddeploy.CustomTargetSourceSHAMetadataKey: clouddeploy.GitCommit,
			},
		}
		fmt.Println("Uploading failed render results")
		rURI, err := r.req.UploadResult(ctx, r.gcsClient, rr)
		if err != nil {
			return fmt.Errorf("error uploading failed render results: %v", err)
		}
		fmt.Printf("Uploaded failed render results to %s\n", rURI)
		return err
	}

	fmt.Println("Uploading render results")
	rURI, err := r.req.UploadResult(ctx, r.gcsClient, res)
	if err != nil {
		return fmt.Errorf("error uploading render results: %v", err)
	}
	fmt.Printf("Uploaded render results to %s\n", rURI)
	return nil
}

// render performs the following steps:
//  1. Generate backend.tf with the GCS backend provided in the params.
//  2. Generate clouddeploy.auto.tfvars with all the variable values provided via TF_VAR_{name} env vars.
//  3. Initialize the Terraform Configuration and validate it.
//  4. Generate speculative Terraform plan and upload it to GCS to use as the Cloud Deploy Release inspector artifact.
//  5. Upload an archived version of the Terraform configuration to GCS so it can be used at deploy time.
//
// Returns either the render results or an error if the render failed.
func (r *renderer) render(ctx context.Context) (*clouddeploy.RenderResult, error) {
	fmt.Printf("Downloading render input archive to %s and unarchiving to %s\n", srcArchivePath, srcPath)
	inURI, err := r.req.DownloadAndUnarchiveInput(ctx, r.gcsClient, srcArchivePath, srcPath)
	if err != nil {
		return nil, fmt.Errorf("unable to download and unarchive render input: %v", err)
	}
	fmt.Printf("Downloaded render input archive from %s\n", inURI)

	// Determine the path to the Terraform configuration. This will be the working directory for Terraform initialization.
	terraformConfigPath := path.Join(srcPath, r.params.configPath)
	if _, err := terraformInit(terraformConfigPath, &terraformInitOptions{}); err != nil {
		return nil, fmt.Errorf("error running terraform init: %v", err)
	}

	backendPath := path.Join(terraformConfigPath, backendFileName)
	fmt.Printf("Generating Terraform backend configuration file: %s\n", backendPath)
	if err := generateBackendFile(backendPath, r.params); err != nil {
		return nil, fmt.Errorf("error generating backend configuration file: %v", err)
	}
	fmt.Printf("Finished generating Terraform backend configuration file: %s\n", backendPath)

	autoVarsPath := path.Join(terraformConfigPath, autoTFVarsFileName)
	fmt.Printf("Generating auto variable definitions file: %s\n", autoVarsPath)
	if err := generateAutoTFVarsFile(autoVarsPath, r.params); err != nil {
		return nil, fmt.Errorf("error generating variable definitions file: %v", err)
	}
	fmt.Printf("Finished generating auto variable definitions file: %s\n", autoVarsPath)

	if _, err := terraformInit(terraformConfigPath, &terraformInitOptions{}); err != nil {
		return nil, fmt.Errorf("error initializing terraform: %v", err)
	}
	if _, err := terraformValidate(terraformConfigPath); err != nil {
		return nil, fmt.Errorf("error validating terraform: %v", err)
	}

	specPlan := []byte{}
	// Only generate the Terraform plan if enabled since this requires the service account to
	// have permissions on the Cloud Storage bucket backend.
	if r.params.enableRenderPlan {
		fmt.Println("Generating speculative Terraform plan for informational purposes")
		if _, err := terraformPlan(terraformConfigPath, speculativePlanFileName); err != nil {
			return nil, fmt.Errorf("error generating terraform plan: %v", err)
		}
		var err error
		specPlan, err = terraformShowPlan(terraformConfigPath, speculativePlanFileName)
		if err != nil {
			return nil, fmt.Errorf("error showing terraform plan: %v", err)
		}
		fmt.Println("Finished generating Terraform plan")
	}

	fmt.Printf("Creating Cloud Deploy Release inspector artifact: %s\n", inspectorArtifactPath)
	if err := createReleaseInspectorArtifact(autoVarsPath, specPlan, inspectorArtifactPath); err != nil {
		return nil, fmt.Errorf("error creating cloud deploy release inspector artifact: %v", err)
	}
	fmt.Println("Uploading Cloud Deploy Release inspector artifact")
	planGCSURI, err := r.req.UploadArtifact(ctx, r.gcsClient, inspectorArtifactName, &gcs.UploadContent{LocalPath: inspectorArtifactPath})
	if err != nil {
		return nil, fmt.Errorf("error uploading speculative plan: %v", err)
	}
	fmt.Printf("Uploaded Cloud Deploy Release inspector artifact to %s\n", planGCSURI)

	// Delete the downloaded providers to save storage space in GCS. The provider versions are stored in the
	// .terraform.lock.hcl file, so the correct versions will be redownloaded at deploy time.
	os.RemoveAll(path.Join(terraformConfigPath, providersDirName))

	// We need to archive all the configuration provided (and generated) instead of just the configuration
	// in the terraformConfigPath in case the Terraform configuration in terraformConfigPath has child modules
	// in a parent directory.
	fmt.Printf("Archiving Terraform configuration in %s for use at deploy time\n", srcPath)
	if err := tarArchiveDir(srcPath, renderedArchiveName); err != nil {
		return nil, fmt.Errorf("error archiving terraform configuration: %v", err)
	}
	fmt.Println("Uploading archived Terraform configuration")
	atURI, err := r.req.UploadArtifact(ctx, r.gcsClient, renderedArchiveName, &gcs.UploadContent{LocalPath: renderedArchiveName})
	if err != nil {
		return nil, fmt.Errorf("error uploading archived terraform configuration: %v", err)
	}
	fmt.Printf("Uploaded archived Terraform configuration to %s\n", atURI)

	renderResult := &clouddeploy.RenderResult{
		ResultStatus: clouddeploy.RenderSucceeded,
		ManifestFile: planGCSURI,
		Metadata: map[string]string{
			clouddeploy.CustomTargetSourceMetadataKey:    tfDeployerSampleName,
			clouddeploy.CustomTargetSourceSHAMetadataKey: clouddeploy.GitCommit,
		},
	}
	return renderResult, nil
}

// generateBackendFile generates a file with a GCS backend configuration at the provided path.
func generateBackendFile(backendPath string, params *params) error {
	// Check whether backend file exists. If it does then fail the render, otherwise create it.
	if _, err := os.Stat(backendPath); !os.IsNotExist(err) {
		return fmt.Errorf("backend configuration file %q already exists, failing render to avoid overwriting any configuration", backendPath)
	}
	backendFile, err := os.Create(backendPath)
	if err != nil {
		return fmt.Errorf("error creating backend configuration file: %v", err)
	}
	defer backendFile.Close()

	hclFile := hclwrite.NewEmptyFile()
	rootBody := hclFile.Body()
	tfBlock := rootBody.AppendNewBlock("terraform", nil)
	tfBlockBody := tfBlock.Body()
	backendBlock := tfBlockBody.AppendNewBlock("backend", []string{"gcs"})
	backendBlockBody := backendBlock.Body()
	backendBlockBody.SetAttributeValue("bucket", cty.StringVal(params.backendBucket))
	backendBlockBody.SetAttributeValue("prefix", cty.StringVal(params.backendPrefix))

	if _, err = backendFile.Write(hclFile.Bytes()); err != nil {
		return fmt.Errorf("error writing to backend configuration file: %v", err)
	}
	return nil
}

// generateAutoTFVarsFile generates a *.auto.tfvars file that contains the variables defined in the environment
// with a "TF_VAR_" prefix and the variables defined in the variable file, if provided. This is done
// so that that the Terraform configuration uploaded at the end of the render has all configuration present for
// a Terraform apply.
func generateAutoTFVarsFile(autoTFVarsPath string, params *params) error {
	// Check whether clouddeploy.auto.tfvars file exists. If it does then fail the render, otherwise create it.
	if _, err := os.Stat(autoTFVarsPath); !os.IsNotExist(err) {
		return fmt.Errorf("cloud deploy auto.tfvars file %q already exists, failing render to avoid overwriting any configuration", autoTFVarsPath)
	}
	autoTFVarsFile, err := os.Create(autoTFVarsPath)
	if err != nil {
		return fmt.Errorf("error creating cloud deploy auto.tfvars file: %v", err)
	}
	defer autoTFVarsFile.Close()

	if len(params.variablePath) > 0 {
		varsPath := path.Join(path.Dir(autoTFVarsPath), params.variablePath)
		fmt.Printf("Attempting to copy contents from %s to %s so the variables are automatically consumed by Terraform\n", varsPath, autoTFVarsPath)
		varsFile, err := os.Open(varsPath)
		if err != nil {
			return fmt.Errorf("unable to open variable file provided at %s: %v", varsPath, err)
		}
		defer varsFile.Close()

		autoTFVarsFile.Write([]byte(fmt.Sprintf("# Sourced from %s.\n", params.variablePath)))
		if _, err := io.Copy(autoTFVarsFile, varsFile); err != nil {
			return fmt.Errorf("unable to copy contents from %s to %s: %v", varsPath, autoTFVarsPath, err)
		}
		autoTFVarsFile.Write([]byte("\n"))
		fmt.Printf("Finished copying contents from %s to %s\n", varsPath, autoTFVarsPath)
	}

	hclFile := hclwrite.NewEmptyFile()
	rootBody := hclFile.Body()

	// Track whether we found any relevant environment variables to determine if we write to the file.
	found := false
	var keys []string
	kv := make(map[string]cty.Value)
	envVars := os.Environ()
	for _, rawEV := range envVars {
		if !strings.HasPrefix(rawEV, "TF_VAR_") {
			continue
		}
		found = true
		fmt.Printf("Found terraform environment variable %s, will add to %s\n", rawEV, autoTFVarsPath)

		// Remove the prefix so we can get the variable name.
		ev := strings.TrimPrefix(rawEV, "TF_VAR_")
		eqIdx := strings.Index(ev, "=")
		// Invalid.
		if eqIdx == -1 {
			continue
		}
		name := ev[:eqIdx]
		rawVal := ev[eqIdx+1:]

		val, err := parseCtyValue(rawVal, name)
		if err != nil {
			return err
		}
		keys = append(keys, name)
		kv[name] = val
	}

	// We sort the entries so the ordering is consistent between Cloud Deploy Releases.
	sort.Strings(keys)
	for _, k := range keys {
		rootBody.SetAttributeValue(k, kv[k])
	}

	if found {
		autoTFVarsFile.Write([]byte("# Sourced from TF_VAR_ prefixed environment variables.\n"))
		if _, err = autoTFVarsFile.Write(hclFile.Bytes()); err != nil {
			return fmt.Errorf("error writing to cloud deploy auto.tfvars file: %v", err)
		}
	}
	return nil
}

// parseCtyValue attempts to parse the provided string value into a cty.Value.
func parseCtyValue(rawVal string, key string) (cty.Value, error) {
	expr, diags := hclsyntax.ParseExpression([]byte(rawVal), "", hcl.InitialPos)
	if diags.HasErrors() {
		return cty.DynamicVal, fmt.Errorf("error parsing %s for variable %s", rawVal, key)
	}

	var val cty.Value
	var valDiags hcl.Diagnostics
	val, valDiags = expr.Value(nil)
	if valDiags.HasErrors() {
		// If extracting the value from the expression fails then it's possible the value is a string (as
		// opposed to a number, list, map, etc), which needs to be in quotes to be properly parsed so we
		// retry with the raw value wrapped in quotes. If this doesn't work then return the initial
		// value error received.
		rawWithQuotes := fmt.Sprintf("%q", rawVal)
		expr, diags := hclsyntax.ParseExpression([]byte(rawWithQuotes), "", hcl.InitialPos)
		if diags.HasErrors() {
			return cty.DynamicVal, fmt.Errorf("error parsing %s for variable %s", rawVal, key)
		}
		var rValDiags hcl.Diagnostics
		val, rValDiags = expr.Value(nil)
		if rValDiags.HasErrors() {
			return cty.DynamicVal, fmt.Errorf("error parsing %s for variable %s", rawVal, key)
		}
	}
	return val, nil
}

// createReleaseInspectorArtifact creates a file that will be returned to Cloud Deploy as the rendered
// manifest so it is viewable in the Release inspector. The file contains the contents of the generated
// variables file and the speculative Terraform plan, if a plan was generated.
func createReleaseInspectorArtifact(autoTFVarsPath string, planData []byte, dstPath string) error {
	dstFile, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("error creating file %s: %v", dstPath, err)
	}
	defer dstFile.Close()

	autoVarsFile, err := os.Open(autoTFVarsPath)
	if err != nil {
		return fmt.Errorf("unable to open generated variable file %s: %v", autoTFVarsPath, err)
	}
	defer autoVarsFile.Close()

	if _, err := io.Copy(dstFile, autoVarsFile); err != nil {
		return fmt.Errorf("unable to copy contents from %s to %s: %v", autoTFVarsPath, dstPath, err)
	}

	// No plan was generated.
	if len(planData) == 0 {
		return nil
	}

	tBytes, err := time.Now().MarshalText()
	if err != nil {
		return fmt.Errorf("unable to marshal currrent time: %v", err)
	}

	dstFile.Write([]byte(fmt.Sprintf("---\n# Speculative Terraform plan generated at %s for informational purposes.\n# This plan is not used when applying the Terraform configuration.\n", string(tBytes))))
	dstFile.Write(planData)
	return nil
}

// tarArchiveDir creates a tar file with the provided name containing all the contents of the provided directory.
func tarArchiveDir(dir string, dst string) error {
	// Determine the sources for the archive, which is all the entries in the directory.
	de, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("unable to read directory contents %s: %v", dir, err)
	}
	var sources []string
	for _, e := range de {
		// Name only returns the final element of the path so we need to reconstruct the path.
		entryPath := path.Join(dir, e.Name())
		sources = append(sources, entryPath)
	}
	return archiver.NewTarGz().Archive(sources, dst)
}
