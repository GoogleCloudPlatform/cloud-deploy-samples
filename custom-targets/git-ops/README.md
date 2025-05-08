# Cloud Deploy Git Deployer Sample

This directory contains a sample implementation of a Cloud Deploy Custom Target
for deploying to a Git repository. The supported Git providers are `github.com`
and `gitlab.com`.

**This is not an officially supported Google product, and it is not covered by a
Google Cloud support contract. To report bugs or request features in a Google
Cloud product, please contact [Google Cloud
support](https://cloud.google.com/support).**

## Overview

The GitOps deployer allows you to use Cloud Deploy to manage your delivery
pipeline while using a Kubernetes git synchronization tool (such as Argo) to
actually apply changes to the cluster. This enables normal operation and use of
Cloud Deploy features (such as progressions, verification, automation, etc…)
with the difference being the deploy job writes the manifest to a git repository
instead of applying to the cluster directly. From there, it is expected that a
git syncing tool is running on the clusters which will synchronize its state.

Example use cases:

*   You have an existing GitOps setup and would like to extend it with Cloud
    Deploy features such as managing the progression sequence, running pre-post
    deployment actions and verifications
*   You have an existing Cloud Deploy pipeline and would like to add git history
    of all changes to the target
*   You want Cloud Deploy to act as your single control pane for all deployment
    activities and want to connect your GitOps flows into it.

## Quickstart

A quickstart that uses this sample is available
[here](./quickstart/QUICKSTART.md).

## Configuration

## Manifest

The Git deployer expects Cloud Deploy to provide a rendered manifest. In other
words, this sample does not implement a custom render and expects Cloud Deploy
to perform its default rendering process.

## Deploy Parameters

| Parameter | Required | Description |
| --- | --- | --- |
| customTarget/gitRepo | Yes | The URI of the Git repository, e.g. "github.com/{owner}/{repository}" |
| customTarget/gitSourceBranch | Yes | The branch used for committing changes |
| customTarget/gitSecret | Yes | The name of the Secret Manager SecretVersion resource used for cloning the Git repository and optionally opening pull requests, e.g. "projects/{project-number}/secrets/{secret-name}/versions/{version-number}" |
| customTarget/gitPath | No | Relative path from the repository root where the manifest will be written. If not provided then defaults to the root of the repository with the file name "manifest.yaml" |
| customTarget/gitUsername | No | The committer username, if not provided then defaults to "Cloud Deploy" |
| customTarget/gitEmail | No | The committer email, if not provided then the email is left empty |
| customTarget/gitCommitMessage | No | The commit message to use, if not provided then defaults to: "Delivery Pipeline: {pipeline-id} Release: {release-id} Rollout: {rollout-id}" |
| customTarget/gitDestinationBranch | No | The branch a pull request will be opened against, if not provided then no pull request is opened and the deploy completes upon the commit and push to the source branch |
| customTarget/gitPullRequestTitle | No | The title of the pull request, if not provided then defaults to "Cloud Deploy: Release {release-id}, Rollout {rollout-id}" |
| customTarget/gitPullRequestBody | No | The body of the pull request, if not provided then defaults to "Project: {project-num} Location: {location} Delivery Pipeline: {pipeline-id} Target: {target-id} Release: {release-id} Rollout: {rollout-id}" |
| customTarget/gitEnablePullRequestMerge | No | Whether to merge the pull request opened against the `gitDestinationBRanch` |
| customTarget/gitEnableArgoSyncPoll | No | Whether to poll the sync status of the Argo Application. The deployer polls the Argo Application until the the merged changes are synced. When enabled the following deploy parameters become required: `gitGKECluster`, `gitArgoApplication`, and `gitArgoNamespace` |
| customTarget/gitGKECluster | No | The name of the GKE cluster hosting the Argo Application resource, required when `gitEnableArgoSyncPoll` is `true` |
| customTarget/gitArgoApplication | No | The name of the Argo Application resource associated with the Git repository, required when `gitEnableArgoSyncPoll` is `true` |
| customTarget/gitArgoNamespace | No | The namespace the Argo Application resource resides in, required when `gitEnableArgoSyncPoll` is `true` |
| customTarget/gitArgoSyncTimeout | No | Duration to poll the sync status of the Argo Application, if not provided then defaults to 30 minutes |

## Secret - Personal Access Token

When using Github, a personal access token must be configured and uploaded to
Secret Manager. When using Gitlab, a project access token can be configured and
uploaded. The service account used in the target execution environment must be
configured with the role `roles/secretmanager.secretAccessor` to read the token
secret from Secret Manager.

The Github PAT must be configured to have `Read and Write` access for `Contents`
and `Pull Requests`.

The Gitlab PAT must be configured to use the role `Maintainer` with the `api`
and `write_repository` permissions.

<a name="build"></a>
# Build the sample image and register a Custom Target Type for Terraform

The `build_and_register.sh` script within this `git-ops` directory can be used
to build the Git deployer image and register a Cloud Deploy custom target type
that references the image. To use the script run the following command:

```shell
./build_and_register.sh -p $PROJECT_ID -r $REGION
```

The script does the following on your behalf:

1.  Create an Artifact Registry Repository
2.  Give the default compute service account access to the Repository
3.  Build the image and push it to the Repository
4.  Create a Cloud Storage bucket and within the bucket a skaffold configuration
    that references the image built
5.  Apply a custom target type for Git to Cloud Deploy that references the
    skaffold configuration in Cloud Storage

## How the sample image works

The Git deployer sample uses the default Cloud Deploy rendering process. After
rendering, deploy operations are handled by this Git deployer.

## Deploy
The deploy process consists of the following steps:

1. Downloaded the rendered manifest generated by Cloud Deploy via the default rendering process.

2. Access the configured Secret Manager SecretVersion.

3. Clone the Git Repository and check out the source branch.

4. Copy the rendered manifest into the source branch then commit and push the changes.

5.  If a destination branch is provided via `customTarget/gitDestinationBranch`:

    a. Open a pull request with the changes from the source branch to the
    destination branch. The pull request is merged if
    `customTarget/gitEnablePullRequestMerge` is `true`.

    b. If `customTarget/gitEnableArgoSyncPoll` is `true` then the deployer polls
    the Argo Application until the status is `Synced` with the merged changes or
    the timeout is reached.

6.  The rendered manifest is uploaded to Cloud Storage as a Cloud Deploy deploy
    artifact.