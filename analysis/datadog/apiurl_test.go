package main

import (
	"testing"
)

func TestSiteToAPIURL_Success(t *testing.T) {
	tests := []struct {
		name    string
		siteURL string
		wantURL string
	}{
		{
			name:    "App prefix",
			siteURL: "https://app.datadoghq.com",
			wantURL: "https://api.datadoghq.com",
		},
		{
			name:    "App prefix and trailing slash",
			siteURL: "https://app.datadoghq.com/",
			wantURL: "https://api.datadoghq.com",
		},
		{
			name:    "US3",
			siteURL: "https://us3.datadoghq.com",
			wantURL: "https://api.us3.datadoghq.com",
		},
		{
			name:    "AP1",
			siteURL: "https://ap1.datadoghq.com",
			wantURL: "https://api.ap1.datadoghq.com",
		},
		{
			name:    "EU1",
			siteURL: "https://app.datadoghq.eu",
			wantURL: "https://api.datadoghq.eu",
		},
		{
			name:    "GOV with app prefix",
			siteURL: "https://app.ddog-gov.com",
			wantURL: "https://api.ddog-gov.com",
		},
		{
			name:    "Hypothetical new site",
			siteURL: "https://us10.datadoghq.com",
			wantURL: "https://api.us10.datadoghq.com",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotURL, err := SiteToAPIURL(tc.siteURL)
			if err != nil {
				t.Errorf("SiteToAPIURL(%q) returned an unexpected error: %v", tc.siteURL, err)
			}
			if gotURL != tc.wantURL {
				t.Errorf("SiteToAPIURL(%q) = %q, want %q", tc.siteURL, gotURL, tc.wantURL)
			}
		})
	}
}

func TestSiteToAPIURL_Failure(t *testing.T) {
	tests := []struct {
		name    string
		siteURL string
	}{
		{
			name: "empty string",
		},
		{
			name:    "unsupported domain",
			siteURL: "https://app.datadoghq.org",
		},
		{
			name:    "random string",
			siteURL: "not-a-url",
		},
		{
			name:    "http instead of https",
			siteURL: "http://app.datadoghq.com",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := SiteToAPIURL(tc.siteURL)
			if err == nil {
				t.Errorf("SiteToAPIURL(%q) succeeded for invalid input, want error", tc.siteURL)
			}
		})
	}
}
