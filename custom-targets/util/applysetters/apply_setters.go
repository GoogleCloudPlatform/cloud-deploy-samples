// Package applysetters is an interface for Skaffold's applysetters package
// to apply kpt-style param transformations for a yaml config file with the
// parameters provided as key value pairs.
package applysetters

import (
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/applysetters"
	"path"
	"sigs.k8s.io/kustomize/kyaml/kio"
)

// ApplyParams sets the value of a kpt-style param in the input file with the values
// from the 'params' map.
func ApplyParams(filePath string, params map[string]string) error {
	s := &applysetters.ApplySetters{}

	addSetters(params, s)

	fileName := path.Base(filePath)
	baseDir := path.Dir(filePath)

	inout := &kio.LocalPackageReadWriter{
		PackagePath:    baseDir,
		NoDeleteFiles:  true,
		MatchFilesGlob: []string{fileName},
	}

	return kio.Pipeline{
		Inputs:  []kio.Reader{inout},
		Filters: []kio.Filter{s},
		Outputs: []kio.Writer{inout},
	}.Execute()
}

// addSetters populates the setter struct with key values provided in params
func addSetters(params map[string]string, fcd *applysetters.ApplySetters) {
	for k, v := range params {
		fcd.Setters = append(fcd.Setters, applysetters.Setter{Name: k, Value: v})
	}
}
