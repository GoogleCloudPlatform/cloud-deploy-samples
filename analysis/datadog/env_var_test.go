package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestValidateEnvVarsValid(t *testing.T) {
	aPISecret := "test-api-secret"
	appSecret := "test-app-secret"
	query1 := "serving.knative.dev/route:foo"
	query2 := "serving.knative.dev/route:bar"
	siteURL := "test-site"
	envVars := []string{
		fmt.Sprintf("DatadogAPISecret=%s", aPISecret),
		fmt.Sprintf("DatadogAppSecret=%s", appSecret),
		fmt.Sprintf("Query_1=%s", query1),
		fmt.Sprintf("Query_2=%s", query2),
		fmt.Sprintf("DatadogURL=%s", siteURL),
	}

	tests := []struct {
		name       string
		envVars    []string
		wantResult *ValidatedEnvVars
	}{
		{
			name:    "Valid environment variables",
			envVars: envVars,
			wantResult: &ValidatedEnvVars{
				APISecret: aPISecret,
				AppSecret: appSecret,
				Queries:   []string{query1, query2},
				SiteURL:   siteURL,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := validateEnvVars(test.envVars)
			if err != nil {
				t.Errorf("validateEnvVars() error = %v", err)
			}

			sort := cmpopts.SortSlices(func(a, b string) bool { return a < b })
			if diff := cmp.Diff(test.wantResult, result, sort); diff != "" {
				t.Errorf("validateEnvVars() mismatch (-want +got):\n%s", diff)
			}

		})
	}
}

func TestValidateEnvVarsInvalid(t *testing.T) {
	tests := []struct {
		name               string
		envVars            []string
		wantErrorSubstring string
	}{
		{
			name:               "Missing Datadog API secret environment variable",
			envVars:            []string{"DatadogAppSecret=test-app-secret", "Query_1=query1", "Query_2=query2", "DatadogURL=test-site"},
			wantErrorSubstring: "missing required environment variable: DatadogAPISecret",
		},
		{
			name:               "Missing Datadog App secret environment variable",
			envVars:            []string{"DatadogAPISecret=test-api-secret", "Query_1=query1", "Query_2=query2", "DatadogURL=test-site"},
			wantErrorSubstring: "missing required environment variable: DatadogAppSecret",
		},
		{
			name:               "Missing Query environment variable",
			envVars:            []string{"DatadogAPISecret=test-secret", "DatadogAppSecret=test-app-secret", "DatadogURL=test-site"},
			wantErrorSubstring: "missing required environment variable: Query",
		},
		{
			name:               "Misspelled Query environment variable so query environment variable is missing",
			envVars:            []string{"DatadogAPISecret=test-secret", "DatadogAppSecret=test-app-secret", "Querry_foo=queryfoo", "DatadogURL=test-site"},
			wantErrorSubstring: "missing required environment variable: Query",
		},
		{
			name:               "Missing Datadog Site environment variable",
			envVars:            []string{"DatadogAPISecret=test-api-secret", "DatadogAppSecret=test-app-secret", "Query_1=query1"},
			wantErrorSubstring: "missing required environment variable: DatadogURL",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := validateEnvVars(test.envVars)
			if err == nil {
				t.Errorf("validateEnvVars() got err = nil, want %v", test.wantErrorSubstring)
			}

			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(test.wantErrorSubstring)) {
				t.Errorf("validateEnvVars() got err = %v, want %v", err, test.wantErrorSubstring)
			}
		})
	}
}
