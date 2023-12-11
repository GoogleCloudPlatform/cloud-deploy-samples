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
	"cloud.google.com/go/storage"
	"context"
	"flag"
	"fmt"
	"github.com/GoogleCloudPlatform/cloud-deploy-samples/custom-targets/util/clouddeploy"
	"os"
)

func main() {
	if err := do(); err != nil {
		fmt.Printf("err: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Done!")
}

func do() error {
	ctx := context.Background()

	gcsClient, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("unable to create gcs client: %v", err)
	}

	flag.BoolVar(&addAliasesMode, "add-aliases-mode", false, "if enabled, adds aliases set in vertexAIAliases environment variable to the deployed model")
	flag.Parse()

	if addAliasesMode {
		ah, err := newAliasHandler(gcsClient)
		if err != nil {
			return fmt.Errorf("unable to create alias handler: %v", err)
		}
		return ah.process(ctx)
	}

	req, err := clouddeploy.DetermineRequest(ctx, gcsClient, []string{"CANARY"})

	if err != nil {
		return err
	}

	params, err := determineParams()

	if err != nil {
		return fmt.Errorf("unable to parse params: %v", err)
	}

	aiPlatformRegion, err := fetchRegionFromModel(params.model)
	if err != nil {
		return fmt.Errorf("unable to parse region from model resource name: %v", err)
	}

	aiPlatformService, err := newAIPlatformService(ctx, aiPlatformRegion)

	if err != nil {
		return fmt.Errorf("unable to create aiplatform.Service object : %v", err)
	}

	handler, err := createRequestHandler(req, params, gcsClient, aiPlatformService)

	if err != nil {
		return fmt.Errorf("unable to create request handler: %v", err)
	}

	return handler.process(ctx)

}
