package main

import (
	"fmt"
	"os"
	"strings"
)

// Environment variable keys specific to the datadog container.
const (
	datadogAPISecretEnvKey = "DatadogAPISecret"
	datadogAppSecretEnvKey = "DatadogAppSecret"
	queryPrefixEnvKey      = "Query"
	datadogLocationEnvKey  = "DatadogLocation"
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
	// Location is the Datadog location to use.
	Location string
}

// checkDuplicates expects environment variables in the k=v format. It
// converts the environment string slice to a map and checks for duplicates
// and malformed entries.
func checkDuplicates(environ []string) (map[string]string, error) {
	envMap := make(map[string]string)

	if len(environ) == 0 {
		return nil, fmt.Errorf("no environment variables found")
	}

	for _, envVar := range environ {
		pair := strings.SplitN(envVar, "=", 2)
		if len(pair) != 2 {
			return nil, fmt.Errorf("incorrect env variable format - expected k=v")
		}

		key := pair[0]
		value := pair[1]
		if key == "" {
			return nil, fmt.Errorf("empty environment variable key")
		}

		if value == "" {
			return nil, fmt.Errorf("empty environment variable value")
		}

		if _, exists := envMap[strings.ToLower(key)]; exists {
			return nil, fmt.Errorf("duplicate environment variable key: %s", key)
		}
		envMap[strings.ToLower(key)] = value
	}
	return envMap, nil
}

// validateEnvVars validates that the required environment variables are set.
func validateEnvVars(environ []string) (*ValidatedEnvVars, error) {
	var apiSecret string
	var appSecret string
	var queries []string
	var location string
	foundAPISecret := false
	foundQuery := false
	foundAppSecret := false

	// Check for duplicate env var keys
	parsedEnv, err := checkDuplicates(environ)
	if err != nil {
		return nil, err
	}

	for key, value := range parsedEnv {
		switch {
		case strings.EqualFold(key, datadogAPISecretEnvKey):
			apiSecret = value
			foundAPISecret = true
		case strings.EqualFold(key, datadogAppSecretEnvKey):
			appSecret = value
			foundAppSecret = true
		case strings.EqualFold(key, datadogLocationEnvKey):
			location = value
		case strings.HasPrefix(strings.ToLower(key), strings.ToLower(queryPrefixEnvKey)):
			queries = append(queries, value)
			foundQuery = true
		default:
			return nil, fmt.Errorf("unknown environment variable: %s", key)
		}
	}

	if !foundAPISecret {
		return nil, fmt.Errorf("missing required environment variable: %s which is used to retrieve the Datadog API key", datadogAPISecretEnvKey)
	}
	if !foundAppSecret {
		return nil, fmt.Errorf("missing required environment variable: %s which is used to retrieve the Datadog App key", datadogAppSecretEnvKey)
	}
	if !foundQuery {
		return nil, fmt.Errorf("missing required environment variable: %s; at least one query is required to call Datadog with", queryPrefixEnvKey)
	}

	return &ValidatedEnvVars{
		APISecret: apiSecret,
		AppSecret: appSecret,
		Queries:   queries,
		Location:  location,
	}, nil
}

// envVars gets the environment variables from the runtime and validates them.
func envVars() (*ValidatedEnvVars, error) {
	environ := os.Environ()
	return validateEnvVars(environ)
}
