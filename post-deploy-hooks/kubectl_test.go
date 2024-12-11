package main

import (
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestKubectlGetArgs(t *testing.T) {
	os.Setenv(releaseEnvKey, "myrelease")
	os.Setenv(pipelineEnvKey, "mypipeline")
	os.Setenv(targetEnvKey, "mytarget")
	os.Setenv(projectEnvKey, "myproject")
	os.Setenv(locationEnvKey, "losangeles")
	labels := "-l deploy.cloud.google.com/delivery-pipeline-id=mypipeline,deploy.cloud.google.com/target-id=mytarget,deploy.cloud.google.com/location=losangeles,deploy.cloud.google.com/project-id=myproject"
	labelsWithRelease := "-l deploy.cloud.google.com/release-id=myrelease,deploy.cloud.google.com/delivery-pipeline-id=mypipeline,deploy.cloud.google.com/target-id=mytarget,deploy.cloud.google.com/location=losangeles,deploy.cloud.google.com/project-id=myproject"

	for _, tc := range []struct {
		name                string
		includeReleaseLabel bool
		resourceType        string
		namespace           string
		wantArgs            []string
	}{
		{
			name:                "basic - no release nor namespace",
			includeReleaseLabel: false,
			resourceType:        "foo",
			wantArgs: []string{
				"get",
				"-o",
				"name",
				labels,
				"foo",
			},
		},
		{
			name:                "with release",
			includeReleaseLabel: true,
			resourceType:        "foo",
			wantArgs: []string{
				"get",
				"-o",
				"name",
				labelsWithRelease,
				"foo",
			},
		},
		{
			name:                "with namespace",
			includeReleaseLabel: false,
			resourceType:        "foo",
			namespace:           "mynamespace",
			wantArgs: []string{
				"get",
				"-o",
				"name",
				labels,
				"--namespace=mynamespace",
				"foo",
			},
		},
		{
			name:                "with namespace and release",
			includeReleaseLabel: true,
			resourceType:        "foo",
			namespace:           "mynamespace",
			wantArgs: []string{
				"get",
				"-o",
				"name",
				labelsWithRelease,
				"--namespace=mynamespace",
				"foo",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gotArgs := kubectlGetArgs(tc.includeReleaseLabel, tc.resourceType, tc.namespace)
			if diff := cmp.Diff(tc.wantArgs, gotArgs); diff != "" {
				t.Errorf("kubectlGetArgs() produced diff (-want, +got):\n%s", diff)
			}
		})
	}
}
