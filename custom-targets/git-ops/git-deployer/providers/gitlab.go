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

// GitLabProvider implements the GitProvider interface for interacting with the Gitlab API.
type GitLabProvider struct {
	Repository string
	Token      string
	Owner      string
}

// gitLabMergeRequest represents the response when querying for a GitLab Merge request.
type gitLabMergeRequest struct {
	InternalID int `json:"iid"`
}

// gitLabMergeResponse represents the response from a GitLab when merging a pull request.
type gitLabMergeResponse struct {
	Sha string `json:"merge_commit_sha"`
}

// OpenPullRequest calls the GitLab API for opening a merge request from a source branch to a destination branch.
func (p *GitLabProvider) OpenPullRequest(src, dst, title, body string) (*PullRequest, error) {
	payload, err := json.Marshal(map[string]string{
		"title":         title,
		"source_branch": src,
		"target_branch": dst,
		"description":   body,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to marshal json for merge request: %v", err)
	}
	reader := bytes.NewReader(payload)
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("https://gitlab.com/api/v4/projects/%s%%2F%s/merge_requests", p.Owner, p.Repository), reader)
	if err != nil {
		return nil, fmt.Errorf("unable to create new request: %v", err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", p.Token))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unable to make request: %v", err)
	}
	defer resp.Body.Close()

	var mr gitLabMergeRequest
	r, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read response body: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("create pull request body: %q, status got: %v want: %v", r, resp.StatusCode, http.StatusCreated)
	}
	if err := json.Unmarshal(r, &mr); err != nil {
		return nil, fmt.Errorf("unable to unmarshal open pull request response: %v", err)
	}

	return &PullRequest{Number: mr.InternalID}, nil
}

// MergePullRequest calls the Gitlab API for merging a merge request.
func (p *GitLabProvider) MergePullRequest(prNo int) (*MergeResponse, error) {
	call := func(prNo int) (*MergeResponse, error) {
		req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("https://gitlab.com/api/v4/projects/%s%%2F%s/merge_requests/%d/merge", p.Owner, p.Repository, prNo), nil)
		if err != nil {
			return nil, fmt.Errorf("unable to create new request: %v", err)
		}

		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", p.Token))

		resp, err := http.DefaultClient.Do(req)
		defer resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("unable to make request: %v", err)
		}
		var mr gitLabMergeResponse
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
		return &MergeResponse{Sha: mr.Sha}, nil
	}

	return mergePullRequestWithRetries(prNo, call)
}
