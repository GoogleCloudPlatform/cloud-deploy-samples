// Package main implements a sample datadog container. It can be used in conjunction with the
// upcoming analysis feature to query datadog for alerts.
// IMPORTANT NOTE: This is a work in progress and not ready for production use.
package main

import (
	"context"
	"fmt"
	"os"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"github.com/GoogleCloudPlatform/cloud-deploy-samples/packages/secrets"
)

// Environment variable keys specific to the datadog container.
const (
	datadogAPISecretEnvKey = "DataDogAPISecret"
)

func main() {
	if err := do(); err != nil {
		fmt.Printf("err: %v\n", err)
		os.Exit(1)
	}
}

func do() error {
	ctx := context.Background()
	// Step 1. Get the environment variables and validate that the required ones were provided.
	secretVersion, found := os.LookupEnv(datadogAPISecretEnvKey)
	if !found {
		fmt.Printf("Required environment variable %s not found. It is required and used to retrieve the Datadog API key. \n", datadogAPISecretEnvKey)
		return fmt.Errorf("required environment variable %s not found", secretVersion)
	}

	// Step 2. Get the secret using the Secret Manager API and the env var they provided.
	smClient, err := secretmanager.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("unable to create secret manager client: %v", err)
	}
	// TODO: b/421427248 - actually use the secret when calling the Datadog API
	_, err = secrets.SecretVersionData(ctx, secretVersion, smClient)
	if err != nil {
		return fmt.Errorf("unable to access datadog secret: %v", err)
	}
	return nil
}
