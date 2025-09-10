package main

import (
	"fmt"
	"strings"
)

// Datadog Site constants
const (
	siteUS1 = "https://api.datadoghq.com"
	siteUS3 = "https://api.us3.datadoghq.com"
	siteUS5 = "https://api.us5.datadoghq.com"
	siteEU1 = "https://api.datadoghq.eu"
	siteAP1 = "https://api.ap1.datadoghq.com"
	siteGov = "https://api.ddog-gov.com"
)

// ddSiteMap maps location codes to Datadog API base URLs.
var ddSiteMap = map[string]string{
	"us1": siteUS1,
	"us3": siteUS3,
	"us5": siteUS5,
	"eu1": siteEU1,
	"ap1": siteAP1,
	"gov": siteGov,
}

// ToSiteURL converts a datadog location string to a site URL.
// It handles case-insensitivity by converting the location to lowercase.
func ToSiteURL(location string) (string, error) {
	site, ok := ddSiteMap[strings.ToLower(location)]
	if !ok {
		return "", fmt.Errorf("unknown Datadog location: %q", location)
	}
	return site, nil
}
