package main

import (
	"fmt"
	"net/url"
	"strings"
)

// SiteToAPIURL converts a datadog site URL string to an API URL. For example,
// "https://app.datadoghq.com" becomes "https://api.datadoghq.com" and
// "https://us1.datadoghq.com" becomes "https://api.us1.datadoghq.com".
func SiteToAPIURL(siteURL string) (string, error) {
	trimmedSiteURL := strings.TrimRight(siteURL, "/")
	u, err := url.Parse(trimmedSiteURL)
	if err != nil {
		return "", fmt.Errorf("unable to parse site URL %q: %w", siteURL, err)
	}
	if u.Scheme != "https" {
		return "", fmt.Errorf("invalid site URL for %q: must be https", siteURL)
	}

	host := u.Hostname()
	if !strings.HasSuffix(host, "datadoghq.com") && !strings.HasSuffix(host, "datadoghq.eu") && !strings.HasSuffix(host, "ddog-gov.com") {
		return "", fmt.Errorf("unknown or unsupported Datadog site URL: %q", siteURL)
	}

	if strings.HasPrefix(trimmedSiteURL, "https://app.") {
		return strings.Replace(trimmedSiteURL, "https://app.", "https://api.", 1), nil
	}
	return strings.Replace(trimmedSiteURL, "https://", "https://api.", 1), nil
}
