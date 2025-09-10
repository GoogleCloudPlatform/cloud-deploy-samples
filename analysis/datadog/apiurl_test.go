package main

import (
	"testing"
)

func TestToSiteURLValid(t *testing.T) {
	tests := []struct {
		name     string
		location string
		wantURL  string
	}{
		{
			name:     "US1 lowercase",
			location: "us1",
			wantURL:  siteUS1,
		},
		{
			name:     "US1 uppercase",
			location: "US1",
			wantURL:  siteUS1,
		},
		{
			name:     "US1 mixed case",
			location: "Us1",
			wantURL:  siteUS1,
		},
		{
			name:     "US3",
			location: "us3",
			wantURL:  siteUS3,
		},
		{
			name:     "US5",
			location: "us5",
			wantURL:  siteUS5,
		},
		{
			name:     "EU1",
			location: "eu1",
			wantURL:  siteEU1,
		},
		{
			name:     "AP1 lowercase",
			location: "ap1",
			wantURL:  siteAP1,
		},
		{
			name:     "AP1 uppercase",
			location: "AP1",
			wantURL:  siteAP1,
		},
		{
			name:     "AP1 mixed case",
			location: "Ap1",
			wantURL:  siteAP1,
		},
		{
			name:     "GOV",
			location: "gov",
			wantURL:  siteGov,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotURL, err := ToSiteURL(tc.location)
			if err != nil {
				t.Fatalf("ToSiteURL(%q) returned an unexpected error: %v", tc.location, err)
			}
			if gotURL != tc.wantURL {
				t.Errorf("ToSiteURL(%q) = %q, want %q", tc.location, gotURL, tc.wantURL)
			}
		})
	}
}

func TestToSiteURLInvalid(t *testing.T) {
	tests := []struct {
		name     string
		location string
		wantErr  string
	}{
		{
			name:     "Inavlid 'us' location",
			location: "us",
			wantErr:  "unknown Datadog location: 'us'",
		},
		{
			name:     "Inavlid empty location",
			location: "",
			wantErr:  "unknown Datadog location: ''",
		},
		{
			name:     "Inavlid 'eu' location",
			location: "eu",
			wantErr:  "unknown Datadog location: 'eu'",
		},
		{
			name:     "Inavlid 'ap' location",
			location: "ap",
			wantErr:  "unknown Datadog location: 'ap'",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ToSiteURL(tc.location)
			if err == nil {
				t.Errorf("ToSiteURL(%q) succeeded for invalid input, want error", tc.location)
			}
		})
	}
}
