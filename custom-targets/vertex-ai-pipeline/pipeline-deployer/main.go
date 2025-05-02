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
	"flag"
	"fmt"
	"os"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/cloud-deploy-samples/custom-targets/util/clouddeploy"
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

	flag.Parse()

	req, err := clouddeploy.DetermineRequest(ctx, gcsClient, []string{"CANARY"})
	if err != nil {
		return err
	}

	params, err := determineParams()
	if err != nil {
		return fmt.Errorf("unable to parse params: %v", err)
	}

	aiPlatformService, err := newAIPlatformService(ctx, params.location)
	if err != nil {
		return fmt.Errorf("unable to create aiplatform.Service object : %v", err)
	}

	handler, err := createRequestHandler(req, params, gcsClient, aiPlatformService)
	if err != nil {
		return fmt.Errorf("unable to create request handler: %v", err)
	}

	return handler.process(ctx)
}
