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

	"cloud.google.com/go/config/apiv1/configpb"
	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/cloud-deploy-samples/custom-targets/util/clouddeploy"
	"github.com/GoogleCloudPlatform/cloud-deploy-samples/packages/gcs"
	"github.com/ghodss/yaml"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/mholt/archiver/v3"
	"github.com/zclconf/go-cty/cty"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	// Path to use when downloading the source input archive file.
	srcArchivePath = "/workspace/archive.tgz"
	// Path to use when unarchiving the source input.
	srcPath = "/workspace/source"
	// File name to use for the generated variables file.
	autoTFVarsFileName = "clouddeploy.auto.tfvars"
	// Name of the file that contains the YAML representation of the Infrastructure Manager Deployment
	// that is applied at deploy time.
	renderedDeploymentFileName = "deployment.yaml"
	// Name of the rendered archive. The rendered archive contains the Terraform configuration after
	// the rendering has completed.
	renderedArchiveName = "terraform-archive.zip"
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
				clouddeploy.CustomTargetSourceMetadataKey:    imDeployerSampleName,
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
//  1. Generate clouddeploy.auto.tfvars with all the variable values provided via imVar_{name} env vars.
//  2. Upload a zip archived version of the Terraform configuration to GCS.
//  3. Upload a YAML representation of the Infrastructure Manager Deployment that will be applied at deploy time to GCS.
//     The Deployment will contain the Terraform configuration zip from (2) as the Terraform Blueprint. This YAML
//     will also be provided to Cloud Deploy as the Release inspector artifact.
//
// Returns either the render results or an error if the render failed.
func (r *renderer) render(ctx context.Context) (*clouddeploy.RenderResult, error) {
	fmt.Printf("Downloading render input archive to %s and unarchiving to %s\n", srcArchivePath, srcPath)
	inURI, err := r.req.DownloadAndUnarchiveInput(ctx, r.gcsClient, srcArchivePath, srcPath)
	if err != nil {
		return nil, fmt.Errorf("unable to download and unarchive render input: %v", err)
	}
	fmt.Printf("Downloaded render input archive from %s\n", inURI)

	// Determine the path to the Terraform configuration.
	terraformConfigPath := path.Join(srcPath, r.params.configPath)
	autoVarsPath := path.Join(terraformConfigPath, autoTFVarsFileName)
	fmt.Printf("Generating auto variable definitions file: %s\n", autoVarsPath)
	if err := generateAutoTFVarsFile(autoVarsPath, r.params); err != nil {
		return nil, fmt.Errorf("error generating variable definitions file: %v", err)
	}
	fmt.Printf("Finished generating auto variable definitions file: %s\n", autoVarsPath)

	// Archive the Terraform configuration into a zip file since this is one of the accepted formats
	// by Infrastructure Manager when updating the Deployment resource with Terraform configuration.
	fmt.Printf("Archiving Terraform configuration in %s into zip file for use at deploy time\n", srcPath)
	if err = zipArchiveDir(terraformConfigPath, renderedArchiveName); err != nil {
		return nil, fmt.Errorf("error archiving terraform configuration: %v", err)
	}
	fmt.Println("Uploading archived Terraform configuration")
	tcURI, err := r.req.UploadArtifact(ctx, r.gcsClient, renderedArchiveName, &gcs.UploadContent{LocalPath: renderedArchiveName})
	if err != nil {
		return nil, fmt.Errorf("error uploading archived terraform configuration: %v", err)
	}
	fmt.Printf("Uploaded archived Terraform configuration to %s\n", tcURI)

	fmt.Println("Creating rendered Deployment for use at deploy time")
	renderedDeploymentYAML, err := r.deploymentYAML(tcURI)
	if err != nil {
		return nil, fmt.Errorf("error creating rendered deployment: %v", err)
	}
	fmt.Println("Uploading rendered Deployment")
	dURI, err := r.req.UploadArtifact(ctx, r.gcsClient, renderedDeploymentFileName, &gcs.UploadContent{Data: renderedDeploymentYAML})
	if err != nil {
		return nil, fmt.Errorf("error uploading rendered deployment: %v", err)
	}
	fmt.Printf("Uploaded rendered Deployment to %s\n", dURI)

	renderResult := &clouddeploy.RenderResult{
		ResultStatus: clouddeploy.RenderSucceeded,
		ManifestFile: dURI,
		Metadata: map[string]string{
			clouddeploy.CustomTargetSourceMetadataKey:    imDeployerSampleName,
			clouddeploy.CustomTargetSourceSHAMetadataKey: clouddeploy.GitCommit,
		},
	}
	return renderResult, nil
}

// deploymentYAML returns the YAML representation of the Infrastructure Manager Deployment that will be applied
// at deploy time based on the Terraform configuration uploaded while rendering, the deploy parameters configured,
// and the render request from Cloud Deploy.
func (r *renderer) deploymentYAML(gcsSourceURI string) ([]byte, error) {
	labels := make(map[string]string)
	if !r.params.disableCloudDeployLabels {
		labels = map[string]string{
			"managed-by":           "google-cloud-deploy",
			"project":              r.req.Project,
			"location":             r.req.Location,
			"delivery-pipeline-id": r.req.Pipeline,
			"release-id":           r.req.Release,
			"target-id":            r.req.Target,
		}
	}

	d := &configpb.Deployment{
		Name:   r.params.deploymentName(),
		Labels: labels,
		Blueprint: &configpb.Deployment_TerraformBlueprint{
			TerraformBlueprint: &configpb.TerraformBlueprint{
				Source: &configpb.TerraformBlueprint_GcsSource{
					GcsSource: gcsSourceURI,
				},
			},
		},
		ImportExistingResources: &r.params.importExistingResources,
	}

	// Use Cloud Deploy workload service account if deploy parameter overwrite wasn't provided.
	serviceAccount := r.params.imServiceAccount
	if len(serviceAccount) == 0 {
		serviceAccount = r.req.WorkloadCBInfo.ServiceAccount
	}
	d.ServiceAccount = &serviceAccount

	// Use Cloud Deploy workload worker pool if present and deploy parameter overwrite wasn't provided.
	if len(r.params.imWorkerPool) != 0 {
		d.WorkerPool = &r.params.imWorkerPool
	} else if len(r.req.WorkloadCBInfo.WorkerPool) != 0 {
		d.WorkerPool = &r.req.WorkloadCBInfo.WorkerPool
	}

	j, err := protojson.Marshal(d)
	if err != nil {
		return nil, fmt.Errorf("error marshaling deployment: %v", err)
	}
	y, err := yaml.JSONToYAML(j)
	if err != nil {
		return nil, fmt.Errorf("error converting deployment json to yaml: %v", err)
	}
	return y, nil
}

// generateAutoTFVarsFile generates a *.auto.tfvars file that contains the variables defined in the
// environment with a "imVar_" prefix and the variables defined in the variable file, if provided.
// This is done so that the Terraform configuration uploaded at the end of the render has all the
// configuration present.
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
		if !strings.HasPrefix(rawEV, imVarEnvKeyPrefix) {
			continue
		}
		found = true
		fmt.Printf("Found infrastucture manager environment variable %s, will add to corresponding variable to %s\n", rawEV, autoTFVarsPath)

		// Remove the prefix so we can get the variable name.
		ev := strings.TrimPrefix(rawEV, imVarEnvKeyPrefix)
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
		autoTFVarsFile.Write([]byte(fmt.Sprintf("# Sourced from %s prefixed deploy parameters.\n", imVarDeployParamKeyPrefix)))
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

// zipArchiveDir creates a zip file with the provided name containing all the contents of the provided directory.
func zipArchiveDir(dir string, dst string) error {
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
	return archiver.NewZip().Archive(sources, dst)
}
