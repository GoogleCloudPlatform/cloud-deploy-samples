/*
Copyright 2023 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package applysetters

import (
	"os"
	"path"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApplySettersFilter(t *testing.T) {
	var tests = []struct {
		name              string
		config            map[string]string
		input             string
		expectedResources string
		errMsg            string
	}{
		{
			name: "set name and label",
			input: `apiVersion: v1
kind: Service
metadata:
  name: "" # from-param: ${app}
---
kind: Service
metadata:
  name: "" # from-param: ${app2}
`,
			config: map[string]string{"app": "my-app", "app2": "myService2"},
			expectedResources: `apiVersion: v1
kind: Service
metadata:
  name: "my-app" # from-param: ${app}
---
kind: Service
metadata:
  name: "myService2" # from-param: ${app2}
`,
		},
	}
	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			baseDir, err := os.MkdirTemp("", "")
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			defer os.RemoveAll(baseDir)

			r, err := os.CreateTemp(baseDir, "k8s-cli-*.yaml")
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			defer os.Remove(r.Name())

			err = os.WriteFile(r.Name(), []byte(test.input), os.ModePerm)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			s := &ApplySetters{}

			Decode(test.config, s)

			fileName := path.Base(r.Name())

			inout := &kio.LocalPackageReadWriter{
				PackagePath:    baseDir,
				NoDeleteFiles:  true,
				MatchFilesGlob: []string{fileName},
			}

			err = kio.Pipeline{
				Inputs:  []kio.Reader{inout},
				Filters: []kio.Filter{s},
				Outputs: []kio.Writer{inout},
			}.Execute()

			if test.errMsg != "" {
				if !assert.NotNil(t, err) {
					t.FailNow()
				}
				if !assert.Contains(t, err.Error(), test.errMsg) {
					t.FailNow()
				}
			}

			if test.errMsg == "" && !assert.NoError(t, err) {
				t.FailNow()
			}

			actualResources, err := os.ReadFile(r.Name())
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			if !assert.Equal(t,
				test.expectedResources,
				string(actualResources)) {
				t.FailNow()
			}
		})
	}
}
