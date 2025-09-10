package main

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestCheckDuplicatesValid(t *testing.T) {
	tests := []struct {
		name     string
		envVars  []string
		wantVars map[string]string
	}{
		{
			name: "Valid environment variables",
			envVars: []string{
				"KEY1=VALUE1",
				"KEY2=VALUE2",
			},
			wantVars: map[string]string{
				"key1": "VALUE1",
				"key2": "VALUE2",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			vars, err := checkDuplicates(test.envVars)
			if err != nil {
				t.Errorf("checkDuplicates() error = %v", err)
			}
			if diff := cmp.Diff(vars, test.wantVars); diff != "" {
				t.Errorf("checkDuplicates() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCheckDuplicatesInvalid(t *testing.T) {
	tests := []struct {
		name    string
		envVars []string
	}{
		{
			name: "Empty environment variables",
			envVars: []string{
				"",
				"",
			},
		},
		{
			name: "Duplicate environment variable with same case",
			envVars: []string{
				"KEY1=VALUE1",
				"KEY1=VALUE2",
			},
		},
		{
			name: "Duplicate environment variable with different cases",
			envVars: []string{
				"KEY1=VALUE1",
				"key1=VALUE2",
			},
		},
		{
			name: "Empty environment variable value",
			envVars: []string{
				"KEY1VALUE1=",
				"KEY2=VALUE2",
			},
		},
		{
			name: "Empty environment variable key",
			envVars: []string{
				"=KEY1VALUE1",
				"KEY2=VALUE2",
			},
		},
		{
			name: "Incorrect env variable format - expected k=v",
			envVars: []string{
				"KEY1VALUE1",
				"KEY2=VALUE2",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := checkDuplicates(test.envVars)
			if err == nil {
				t.Errorf("checkDuplicates() error = nil, want error")
			}
		})
	}
}

func TestValidateEnvVarsValid(t *testing.T) {
	testAPISecret := "test-api-secret"
	testAppSecret := "test-app-secret"
	testQueries := []string{"serving.knative.dev/route:foo", "serving.knative.dev/route:bar"}
	testLocation := "us-central1"
	testEnvVarsWithLocation := []string{
		"DatadogAPISecret=test-api-secret",
		"DatadogAppSecret=test-app-secret",
		"Query_1=serving.knative.dev/route:foo",
		"Query_2=serving.knative.dev/route:bar",
		"DatadogLocation=us-central1",
	}
	testEnvVarsWithoutLocation := []string{
		"DatadogAPISecret=test-api-secret",
		"DatadogAppSecret=test-app-secret",
		"Query_1=serving.knative.dev/route:foo",
		"Query_2=serving.knative.dev/route:bar",
	}

	tests := []struct {
		name       string
		envVars    []string
		wantResult *ValidatedEnvVars
	}{
		{
			name:    "Valid environment variables with location defined",
			envVars: testEnvVarsWithLocation,
			wantResult: &ValidatedEnvVars{
				APISecret: testAPISecret,
				AppSecret: testAppSecret,
				Queries:   testQueries,
				Location:  testLocation,
			},
		},
		{
			name:    "Valid environment variables without location defined",
			envVars: testEnvVarsWithoutLocation,
			wantResult: &ValidatedEnvVars{
				APISecret: testAPISecret,
				AppSecret: testAppSecret,
				Queries:   testQueries,
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
			envVars:            []string{"DatadogAppSecret=test-app-secret", "Query_1=query1", "Query_2=query2"},
			wantErrorSubstring: "missing required environment variable: DatadogAPISecret",
		},
		{
			name:               "Missing Datadog App secret environment variable",
			envVars:            []string{"DatadogAPISecret=test-api-secret", "Query_1=query1", "Query_2=query2"},
			wantErrorSubstring: "missing required environment variable: DatadogAppSecret",
		},
		{
			name:               "Missing Query environment variable",
			envVars:            []string{"DatadogAPISecret=test-secret", "DatadogAppSecret=test-app-secret"},
			wantErrorSubstring: "missing required environment variable: Query",
		},
		{
			name:               "Mispelled Query environment variable",
			envVars:            []string{"DatadogAPISecret=test-secret", "DatadogAppSecret=test-app-secret", "Querry_foo=queryfoo"},
			wantErrorSubstring: "unknown environment variable: Querry_foo",
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
