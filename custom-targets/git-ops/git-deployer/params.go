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

package main

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Environment variable keys whose values determine the behavior of the Git deployer.
// Cloud Deploy transforms a deploy parameter "customTarget/gitRepo" into an
// environment variable of the form "CLOUD_DEPLOY_customTarget_gitRepo".
const (
	gitRepoEnvKey               = "CLOUD_DEPLOY_customTarget_gitRepo"
	gitPathEnvKey               = "CLOUD_DEPLOY_customTarget_gitPath"
	gitSourceBranchEnvKey       = "CLOUD_DEPLOY_customTarget_gitSourceBranch"
	gitSecretEnvKey             = "CLOUD_DEPLOY_customTarget_gitSecret"
	gitUsernameEnvKey           = "CLOUD_DEPLOY_customTarget_gitUsername"
	gitEmailEnvKey              = "CLOUD_DEPLOY_customTarget_gitEmail"
	gitCommitMessageEnvKey      = "CLOUD_DEPLOY_customTarget_gitCommitMessage"
	gitDestinationBranchEnvKey  = "CLOUD_DEPLOY_customTarget_gitDestinationBranch"
	gitPullRequestTitleEnvKey   = "CLOUD_DEPLOY_customTarget_gitPullRequestTitle"
	gitPullRequestBodyEnvKey    = "CLOUD_DEPLOY_customTarget_gitPullRequestBody"
	gitEnableArgoSyncPollEnvKey = "CLOUD_DEPLOY_customTarget_gitEnableArgoSyncPoll"
	gitGKEClusterEnvKey         = "CLOUD_DEPLOY_customTarget_gitGKECluster"
	gitArgoAppEnvKey            = "CLOUD_DEPLOY_customTarget_gitArgoApplication"
	gitArgoNamespaceEnvKey      = "CLOUD_DEPLOY_customTarget_gitArgoNamespace"
	gitArgoSyncTimeoutEnvKey    = "CLOUD_DEPLOY_customTarget_gitArgoSyncTimeout"
)

const (
	// Default timeout to use when polling the sync status of the Argo application.
	defaultSyncTimeout = 30 * time.Minute
)

type params struct {
	// The URI of the Git repository, e.g. "github.com/{owner}/{repository}".
	gitRepo string
	// Relative path from the repository root where the manifest will be written. If not provided
	// then defaults to the root of the repository with file name "manifest.yaml".
	gitPath string
	// The branch used for committing changes.
	gitSourceBranch string
	// The name of the Secret Manager SecretVersion resource used for cloning the Git repository
	// and optionally opening pull requests.
	gitSecret string
	// The committer username. If not provided then defaults to "Cloud Deploy".
	gitUsername string
	// The commiter email. If not provided then the email address is left empty.
	gitEmail string
	// The commit message to use. If not provided then defaults to:
	// "Delivery Pipeline: {pipeline-id} Release: {release-id} Rollout: {rollout-id}"
	gitCommitMessage string
	// The branch a pull request will be opened against. If not provided then no pull request is
	// opened and the deploy completes upon the commit and push to the git source branch.
	gitDestinationBranch string
	// The title of the pull request. If not provided then defaults to:
	// "Cloud Deploy: Release {release-id}, Rollout {rollout-id}"
	gitPullRequestTitle string
	// The body of the pull request. If not provided then defaults to:
	// "Project: {project-num}
	//  Location: {location}
	// 	Delivery Pipeline: {pipeline-id}
	//  Target: {target-id}
	//	Release: {release-id}
	//	Rollout: {rollout-id}"
	gitPullRequestBody string
	// Whether to poll the sync status of an Argo Application. If enabled then the deploy only
	// succeeds if the Argo Application is synced with the committed changes.
	enableArgoSyncPoll bool
	// The name of the GKE cluster hosting the Argo Application resource.
	gkeCluster string
	// The name of the Argo Application resource associated with the Git repository.
	argoApp string
	// The namespace the Argo Application resource resides in.
	argoNamespace string
	// Duration to poll the sync status of the Argo application. If not provided then defaults to
	// 30 minutes.
	argoSyncTimeout time.Duration
}

// determineParams returns the params provided in the execution environment via environment variables.
func determineParams() (*params, error) {
	params := &params{}
	// Required parameters:
	repo := os.Getenv(gitRepoEnvKey)
	if len(repo) == 0 {
		return nil, fmt.Errorf("parameter %q is required", gitRepoEnvKey)
	}
	params.gitRepo = repo

	secret := os.Getenv(gitSecretEnvKey)
	if len(secret) == 0 {
		return nil, fmt.Errorf("parameter %q is required", gitSecretEnvKey)
	}
	params.gitSecret = secret

	srcBranch := os.Getenv(gitSourceBranchEnvKey)
	if len(srcBranch) == 0 {
		return nil, fmt.Errorf("parameter %q is required", gitSourceBranchEnvKey)
	}
	params.gitSourceBranch = srcBranch

	// Optional parameters:
	params.gitPath = os.Getenv(gitPathEnvKey)
	params.gitUsername = os.Getenv(gitUsernameEnvKey)
	params.gitEmail = os.Getenv(gitEmailEnvKey)
	params.gitCommitMessage = os.Getenv(gitCommitMessageEnvKey)
	params.gitDestinationBranch = os.Getenv(gitDestinationBranchEnvKey)
	params.gitPullRequestTitle = os.Getenv(gitPullRequestTitleEnvKey)
	params.gitPullRequestBody = os.Getenv(gitPullRequestBodyEnvKey)

	enableSync := false
	es, ok := os.LookupEnv(gitEnableArgoSyncPollEnvKey)
	if ok {
		var err error
		enableSync, err = strconv.ParseBool(es)
		if err != nil {
			return nil, fmt.Errorf("failed to parse parameter %q: %v", gitEnableArgoSyncPollEnvKey, err)
		}
	}
	params.enableArgoSyncPoll = enableSync

	if enableSync {
		// If Argo sync is enabled then some additional parameters become required:
		gkeCluster := os.Getenv(gitGKEClusterEnvKey)
		if len(gkeCluster) == 0 {
			return nil, fmt.Errorf("parameter %q is required when Argo sync polling is enabled", gitGKEClusterEnvKey)
		}
		params.gkeCluster = gkeCluster

		argoApp := os.Getenv(gitArgoAppEnvKey)
		if len(argoApp) == 0 {
			return nil, fmt.Errorf("parameter %q is required when Argo sync polling is enabled", gitArgoAppEnvKey)
		}
		params.argoApp = argoApp

		argoNamespace := os.Getenv(gitArgoNamespaceEnvKey)
		if len(argoNamespace) == 0 {
			return nil, fmt.Errorf("parameter %q is required when Argo sync polling is enabled", gitArgoNamespaceEnvKey)
		}
		params.argoNamespace = argoNamespace

		// Optional Argo sync parameters:
		syncTimeout := defaultSyncTimeout
		st := os.Getenv(gitArgoSyncTimeoutEnvKey)
		if len(st) != 0 {
			var err error
			syncTimeout, err = time.ParseDuration(st)
			if err != nil {
				return nil, fmt.Errorf("failed to parse parameter %q: %v", gitArgoSyncTimeoutEnvKey, err)
			}
		}
		params.argoSyncTimeout = syncTimeout
	}

	return params, nil
}
