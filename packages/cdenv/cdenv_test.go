package cdenv

import (
	"testing"

	"github.com/google/go-cmp/cmp"
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
			vars, err := CheckDuplicates(test.envVars)
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
			_, err := CheckDuplicates(test.envVars)
			if err == nil {
				t.Errorf("checkDuplicates() error = nil, want error")
			}
		})
	}
}
