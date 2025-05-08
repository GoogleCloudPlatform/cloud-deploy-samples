// Copyright 2023 Google LLC

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     https://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// GitHubProvider implements the GitProvider interface for interacting with the Github API.
type GitHubProvider struct {
	Repository string
	Token      string
	Owner      string
}

// OpenPullRequest calls the GitHub API for opening a pull request from a source branch to a destination branch.
func (p *GitHubProvider) OpenPullRequest(src, dst, title, body string) (*PullRequest, error) {
	payload, err := json.Marshal(map[string]string{
		"title": title,
		"head":  src,
		"base":  dst,
		"body":  body,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to marshal json for pull request: %v", err)
	}
	reader := bytes.NewReader(payload)
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls", p.Owner, p.Repository), reader)
	if err != nil {
		return nil, fmt.Errorf("unable to create new request: %v", err)
	}

	req.Header.Add("Accept", "application/vnd.github+json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", p.Token))
	req.Header.Add("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unable to make request: %v", err)
	}
	defer resp.Body.Close()
	var pr PullRequest
	r, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read response body: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("create pull request body: %q, status got: %v want: %v", r, resp.StatusCode, http.StatusCreated)
	}
	if err := json.Unmarshal(r, &pr); err != nil {
		return nil, fmt.Errorf("unable to unmarshal open pull request response: %v", err)
	}

	return &pr, nil
}

// MergePullRequest calls the GitHub API for merging a pull request.
func (p *GitHubProvider) MergePullRequest(prNo int) (*MergeResponse, error) {
	call := func(prNo int) (*MergeResponse, error) {
		payload, err := json.Marshal(map[string]string{
			"merge_method": "merge",
		})
		if err != nil {
			return nil, fmt.Errorf("unable to marshal json for merging pull request: %v", err)
		}
		reader := bytes.NewReader(payload)
		req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls/%d/merge", p.Owner, p.Repository, prNo), reader)
		if err != nil {
			return nil, fmt.Errorf("unable to create new request: %v", err)
		}

		req.Header.Add("Accept", "application/vnd.github+json")
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", p.Token))
		req.Header.Add("X-GitHub-Api-Version", "2022-11-28")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("unable to make request: %v", err)
		}
		defer resp.Body.Close()

		var mr MergeResponse
		r, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("unable to read response body: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("merge pull request body: %q, status got: %v want: %v", r, resp.StatusCode, http.StatusOK)
		}
		if err := json.Unmarshal(r, &mr); err != nil {
			return nil, fmt.Errorf("unable to unmarshal merge pull request response: %v", err)
		}

		return &mr, nil
	}

	return mergePullRequestWithRetries(prNo, call)
}
