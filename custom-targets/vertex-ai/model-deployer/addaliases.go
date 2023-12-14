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

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/cloud-deploy-samples/custom-targets/util/clouddeploy"
	"google.golang.org/api/aiplatform/v1"
	cdapi "google.golang.org/api/clouddeploy/v1"
)

// aliasAssigner is responsible for applying model aliases during a post-deploy operation.

type aliasAssigner struct {
	gcsClient *storage.Client
	request   *addAliasesRequest
}

// process applies model aliases during a post-deploy operation.
func (aa aliasAssigner) process(ctx context.Context) error {
	cdService, err := cdapi.NewService(ctx)
	if err != nil {
		return fmt.Errorf("unable to create cloud deploy API service: %v", err)
	}

	releaseName := fmt.Sprintf("projects/%s/locations/%s/deliveryPipelines/%s/releases/%s", aa.request.project, aa.request.location, aa.request.pipeline, aa.request.release)

	release, err := cdService.Projects.Locations.DeliveryPipelines.Releases.Get(releaseName).Do()
	if err != nil {
		return fmt.Errorf("unable to fetch release to determine location of rendered manifest: %v", err)
	}

	ta, ok := release.TargetArtifacts[aa.request.target]
	if !ok {
		return fmt.Errorf("target artifact does not exist in release")
	}

	pa, ok := ta.PhaseArtifacts[aa.request.phase]
	if !ok {
		return fmt.Errorf("target phase artifact not found in release")
	}

	manifestGcsPath := fmt.Sprintf("%s/%s", ta.ArtifactUri, pa.ManifestPath)
	localManifest := "manifest.yaml"
	fmt.Printf("Downloading deploy input manifest from %q.\n", manifestGcsPath)

	deployRequest := &clouddeploy.DeployRequest{
		ManifestGCSPath: manifestGcsPath,
	}

	fmt.Printf("Downloading rendered manifest.\n")
	if _, err := deployRequest.DownloadManifest(ctx, aa.gcsClient, localManifest); err != nil {
		fmt.Println("Failed to download rendered manifest.")
		return fmt.Errorf("failed to download local manifest: %v", err)
	}

	deployedModelRequest, err := deployModelFromManifest(localManifest)
	if err != nil {
		return err
	}

	modelName := deployedModelRequest.DeployedModel.Model

	modelRegion, err := regionFromModel(modelName)
	if err != nil {
		return fmt.Errorf("unable to obtain region where deployed model is located: %v", err)
	}

	aiPlatformService, err := newAIPlatformService(ctx, modelRegion)
	if err != nil {
		return fmt.Errorf("unable to create aiplatform service: %v", err)
	}

	mergeVersionAliasRequest := &aiplatform.GoogleCloudAiplatformV1MergeVersionAliasesRequest{VersionAliases: aa.request.aliases}
	updatedModel, err := aiPlatformService.Projects.Locations.Models.MergeVersionAliases(modelName, mergeVersionAliasRequest).Do()
	if err != nil {
		return fmt.Errorf("unable to update model version aliases")
	}

	fmt.Printf("Successfully applied new aliases: %s. Current aliases are: %s\n", aa.request.aliases, updatedModel.VersionAliases)

	return nil

}
