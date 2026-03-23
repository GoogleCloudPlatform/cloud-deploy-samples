package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/GoogleCloudPlatform/cloud-deploy-samples/packages/cdenv"
)

// Environment variable keys specific to the datadog container.
const (
	datadogAPISecretEnvKey = "DatadogAPISecret"
	datadogAppSecretEnvKey = "DatadogAppSecret"
	queryPrefixEnvKey      = "Query"
	datadogURLEnvKey       = "DatadogURL"
)

// ValidatedEnvVars holds the validated environment variable values.
type ValidatedEnvVars struct {
	// APISecret is a GCP Secret Version used to store the Datadog API key.
	// The value will look like "projects/{project-number}/secrets/{secret-name}/versions/{version-number}".
	APISecret string
	// AppSecret is a GCP Secret Version used to store the Datadog App key.
	// The value will look like "projects/{project-number}/secrets/{secret-name}/versions/{version-number}".
	AppSecret string
	// Queries is a list of Datadog queries to execute.
	Queries []string
	// SiteURL is the Datadog site URL to use. For example https://us5.datadoghq.com.
	// For a full list, see here https://docs.datadoghq.com/getting_started/site/#access-the-datadog-site"
	SiteURL string
}

// validateEnvVars validates that the required environment variables are set.
func validateEnvVars(environ []string) (*ValidatedEnvVars, error) {
	var apiSecret string
	var appSecret string
	var queries []string
	var siteURL string

	// Check for duplicate env var keys
	parsedEnv, err := cdenv.CheckDuplicates(environ)
	if err != nil {
		return nil, err
	}

	for key, value := range parsedEnv {
		switch {
		case strings.EqualFold(key, datadogAPISecretEnvKey):
			apiSecret = value
		case strings.EqualFold(key, datadogAppSecretEnvKey):
			appSecret = value
		case strings.EqualFold(key, datadogURLEnvKey):
			siteURL = value
		case strings.HasPrefix(strings.ToLower(key), strings.ToLower(queryPrefixEnvKey)):
			queries = append(queries, value)
		default:
			// Cloud Deploy sends other environment variables, ignore any unknowns and move on.
			continue
		}
	}

	if apiSecret == "" {
		return nil, fmt.Errorf("missing required environment variable: %s which is used to retrieve the Datadog API key", datadogAPISecretEnvKey)
	}
	if appSecret == "" {
		return nil, fmt.Errorf("missing required environment variable: %s which is used to retrieve the Datadog App key", datadogAppSecretEnvKey)
	}
	if len(queries) == 0 {
		return nil, fmt.Errorf("missing required environment variable: %s; at least one query is required to call Datadog with", queryPrefixEnvKey)
	}
	if siteURL == "" {
		return nil, fmt.Errorf("missing required environment variable: %s which is used to identify the Datadog site to use", datadogURLEnvKey)
	}

	return &ValidatedEnvVars{
		APISecret: apiSecret,
		AppSecret: appSecret,
		SiteURL:   siteURL,
		Queries:   queries,
	}, nil
}

// envVars gets the environment variables from the runtime and validates them.
func envVars() (*ValidatedEnvVars, error) {
	environ := os.Environ()
	return validateEnvVars(environ)
}
