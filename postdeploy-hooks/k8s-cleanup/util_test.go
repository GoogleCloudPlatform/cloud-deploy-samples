package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDiffSlices(t *testing.T) {

	for _, tc := range []struct {
		name     string
		slice1   []string
		slice2   []string
		wantDiff []string
	}{
		{
			name:   "no diff",
			slice1: []string{"1", "2", "3"},
			slice2: []string{"1", "2", "3"},
		},
		{
			name:     "no overlap",
			slice1:   []string{"1", "2", "3"},
			slice2:   []string{"4", "5", "6"},
			wantDiff: []string{"1", "2", "3"},
		},
		{
			name:     "some overlap",
			slice1:   []string{"1", "2", "3", "4"},
			slice2:   []string{"1", "4", "5", "6"},
			wantDiff: []string{"2", "3"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gotDiff := diffSlices(tc.slice1, tc.slice2)
			if diff := cmp.Diff(tc.wantDiff, gotDiff); diff != "" {
				t.Errorf("diffSlices() produced diff (-want, +got):\n%s", diff)
			}
		})
	}
}
