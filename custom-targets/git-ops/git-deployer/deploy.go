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
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/storage"
	provider "github.com/GoogleCloudPlatform/cloud-deploy-samples/custom-targets/git-ops/git-deployer/providers"
	"github.com/GoogleCloudPlatform/cloud-deploy-samples/custom-targets/util/clouddeploy"
	"github.com/GoogleCloudPlatform/cloud-deploy-samples/packages/secrets"
)

const (
	// Argo Application custom resource type.
	argoCRType = "application"
	// Argo Synced status.
	argoSyncedStatus = "Synced"
	// Argo sync interval is how often to poll the Argo Application for the sync status.
	argoSyncInterval = 15 * time.Second
)

// deployer implements the requestHandler interface for deploy requests.
type deployer struct {
	req       *clouddeploy.DeployRequest
	params    *params
	gcsClient *storage.Client
	smClient  *secretmanager.Client
}

// process processes a deploy request and uploads succeeded or failed results to GCS for Cloud Deploy.
func (d *deployer) process(ctx context.Context) error {
	fmt.Println("Processing deploy request")

	res, err := d.deploy(ctx)
	if err != nil {
		fmt.Printf("Deploy failed: %v\n", err)
		dr := &clouddeploy.DeployResult{
			ResultStatus:   clouddeploy.DeployFailed,
			FailureMessage: err.Error(),
			Metadata: map[string]string{
				clouddeploy.CustomTargetSourceMetadataKey:    gitDeployerSampleName,
				clouddeploy.CustomTargetSourceSHAMetadataKey: clouddeploy.GitCommit,
			},
		}
		fmt.Println("Uploading failed deploy results")
		rURI, err := d.req.UploadResult(ctx, d.gcsClient, dr)
		if err != nil {
			return fmt.Errorf("error uploading failed deploy results: %v", err)
		}
		fmt.Printf("Uploaded failed deploy results to %s\n", rURI)
		return err
	}

	fmt.Println("Uploading deploy results")
	rURI, err := d.req.UploadResult(ctx, d.gcsClient, res)
	if err != nil {
		return fmt.Errorf("error uploading deploy results: %v", err)
	}
	fmt.Printf("Uploaded deploy results to %s\n", rURI)
	return nil
}

// deploy performs the following steps:
//  1. Access the configured Secret Manager SecretVersion.
//  2. Clone the Git Repository and check out the configured source branch.
//  3. Copy the rendered manifest into the source branch, commit, and push the changes.
//  4. If a destination branch is configured:
//     a. Open a pull request with the changes from the source branch to the destination branch.
//     b. If Argo sync polling is enabled then merge the pull request and poll the Argo application
//     until the status is Synced.
func (d *deployer) deploy(ctx context.Context) (*clouddeploy.DeployResult, error) {
	secret, err := secrets.SecretVersionData(ctx, d.params.gitSecret, d.smClient)
	if err != nil {
		return nil, fmt.Errorf("unable to access git secret: %v", err)
	}

	repoParts := strings.Split(d.params.gitRepo, "/")
	if len(repoParts) != 3 {
		return nil, fmt.Errorf("invalid git repository reference: %q", d.params.gitRepo)
	}
	hostname, owner, repoName := repoParts[0], repoParts[1], repoParts[2]
	gitRepo := newGitRepository(hostname, owner, repoName, d.params.gitEmail, d.params.gitUsername)
	if err := d.setupGitWorkspace(ctx, secret, gitRepo); err != nil {
		return nil, fmt.Errorf("unable to set up git workspace: %v", err)
	}

	localManifest := "manifest.yaml"
	fmt.Printf("Downloading rendered manifest to %s\n", localManifest)
	mURI, err := d.req.DownloadManifest(ctx, d.gcsClient, localManifest)
	if err != nil {
		return nil, fmt.Errorf("unable to download rendered manifest: %v", err)
	}
	fmt.Printf("Downloaded rendered manifest from %s\n", mURI)

	fmt.Println("Copying rendered manifest into local Git repository")
	gitManifestPath, err := copyToLocalGitRepo(localManifest, repoName, d.params.gitPath)
	if err != nil {
		return nil, fmt.Errorf("unable to copy manifest to local git repository: %v", err)
	}
	op, err := gitRepo.detectDiff()
	if err != nil {
		return nil, fmt.Errorf("unable to run git status: %v", err)
	}
	if len(op) == 0 {
		return nil, fmt.Errorf("no diff detected between the rendered manifest and the manifest on branch %s", d.params.gitSourceBranch)
	}
	fmt.Printf("Committing and pushing changes to branch %s\n", d.params.gitSourceBranch)
	if err := d.commitPushGitWorkspace(ctx, gitRepo); err != nil {
		return nil, fmt.Errorf("unable to commit and push changes: %v", err)
	}

	if err := d.handleDestinationBranch(ctx, gitRepo, secret); err != nil {
		return nil, err
	}

	fmt.Println("Uploading rendered manifest as a deploy artifact")
	dURI, err := d.req.UploadArtifact(ctx, d.gcsClient, "manifest.yaml", &clouddeploy.GCSUploadContent{LocalPath: gitManifestPath})
	if err != nil {
		return nil, fmt.Errorf("error uploading deploy artifact: %v", err)
	}
	fmt.Printf("Uploaded deploy artifact to %s\n", dURI)

	return &clouddeploy.DeployResult{
		ResultStatus:  clouddeploy.DeploySucceeded,
		ArtifactFiles: []string{dURI},
		Metadata: map[string]string{
			clouddeploy.CustomTargetSourceMetadataKey:    gitDeployerSampleName,
			clouddeploy.CustomTargetSourceSHAMetadataKey: clouddeploy.GitCommit,
		},
	}, nil
}

// setupGitWorkspace clones the Git repository and checks out the configured source branch.
func (d *deployer) setupGitWorkspace(ctx context.Context, secret string, gitRepo *gitRepository) error {
	fmt.Printf("Cloning Git repository %s\n", d.params.gitRepo)
	if _, err := gitRepo.cloneRepo(secret); err != nil {
		return fmt.Errorf("failed to clone git repository %s: %v", d.params.gitRepo, err)
	}
	if err := gitRepo.config(); err != nil {
		return fmt.Errorf("failed setting up the git config in the git repository: %v", err)
	}
	fmt.Printf("Checking out branch %s\n", d.params.gitSourceBranch)
	if _, err := gitRepo.checkoutBranch(d.params.gitSourceBranch); err != nil {
		return fmt.Errorf("unable to checkout branch %s: %v", d.params.gitSourceBranch, err)
	}
	output, err := gitRepo.checkIfExists(d.params.gitSourceBranch)
	if err != nil {
		return fmt.Errorf("unable to check if branch %s exists: %v", d.params.gitSourceBranch, err)
	}
	if output != nil {
		if _, err := gitRepo.pull(d.params.gitSourceBranch); err != nil {
			return fmt.Errorf("unable to pull branch %s: %v", d.params.gitSourceBranch, err)
		}
	}
	return nil
}

// commitPushGitWorkspace commits and pushes changes in the local Git workspace to the source branch.
func (d *deployer) commitPushGitWorkspace(ctx context.Context, gitRepo *gitRepository) error {
	if _, err := gitRepo.add(); err != nil {
		return fmt.Errorf("unable to git add changes: %v", err)
	}
	commitMsg := d.params.gitCommitMessage
	if len(commitMsg) == 0 {
		commitMsg = fmt.Sprintf("Delivery Pipeline: %s Release: %s Rollout: %s", d.req.Pipeline, d.req.Release, d.req.Rollout)
	}
	if _, err := gitRepo.commit(commitMsg); err != nil {
		return fmt.Errorf("unable to git commit changes: %v", err)
	}
	if _, err := gitRepo.push(d.params.gitSourceBranch); err != nil {
		return fmt.Errorf("unable to git push changes to branch %s: %v", d.params.gitSourceBranch, err)
	}
	return nil
}

// handleDestinationBranch opens a pull request on the destination branch if provided and will optionally
// merge the PR if configured. Additionally, if Argo sync polling is enabled then the status of the Argo
// Application is polled until it's synced.
func (d *deployer) handleDestinationBranch(ctx context.Context, gitRepo *gitRepository, secret string) error {
	// If no destination branch is provided then there is no need to open a pull request.
	if len(d.params.gitDestinationBranch) == 0 {
		return nil
	}

	title := d.params.gitPullRequestTitle
	if len(title) == 0 {
		title = fmt.Sprintf("Cloud Deploy: Release %s, Rollout %s", d.req.Release, d.req.Rollout)
	}
	body := d.params.gitPullRequestBody
	if len(body) == 0 {
		body = fmt.Sprintf("Project: %s\nLocation: %s\nDelivery Pipeline: %s\nTarget: %s\nRelease: %s\nRollout: %s",
			d.req.Project,
			d.req.Location,
			d.req.Pipeline,
			d.req.Target,
			d.req.Release,
			d.req.Rollout,
		)
	}

	gitProvider, err := provider.CreateProvider(gitRepo.hostname, gitRepo.repoName, gitRepo.owner, secret)
	if err != nil {
		return fmt.Errorf("unable to create git provider: %v", err)
	}
	fmt.Printf("Opening pull request from %s to %s\n", d.params.gitSourceBranch, d.params.gitDestinationBranch)
	pr, err := gitProvider.OpenPullRequest(d.params.gitSourceBranch, d.params.gitDestinationBranch, title, body)
	if err != nil {
		return fmt.Errorf("unable to open pull request from %s to %s: %v", d.params.gitSourceBranch, d.params.gitDestinationBranch, err)
	}

	if !d.params.enablePullRequestMerge {
		return nil
	}
	fmt.Println("Merging the pull request")
	mr, err := gitProvider.MergePullRequest(pr.Number)
	if err != nil {
		return fmt.Errorf("unable to merge pull request %d: %v", pr.Number, err)
	}

	if !d.params.enableArgoSyncPoll {
		return nil
	}
	fmt.Printf("Argo sync polling is enabled, setting up cluster credentials for %s\n", d.params.gkeCluster)
	if _, err := gcloudClusterCredentials(d.params.gkeCluster); err != nil {
		return fmt.Errorf("unable to set up cluster credentials: %v", err)
	}
	fmt.Printf("Checking for the existence of the Argo Application %s in namespace %s\n", d.params.argoApp, d.params.argoNamespace)
	if _, err := verifyResourceExists(argoCRType, d.params.argoApp, d.params.argoNamespace); err != nil {
		return fmt.Errorf("argo application custom resource not found: %v", err)
	}

	fmt.Println("Polling Argo Application until it's synced with the merged changes")
	if err := pollSyncStatus(d.params.argoApp, d.params.argoNamespace, mr.Sha, d.params.argoSyncTimeout); err != nil {
		return fmt.Errorf("unable to verify argo application is synced: %v", err)
	}
	fmt.Printf("Argo Application synced with the merged changes\n")
	return nil
}

// copyToLocalGitRepo copies a local file to a local Git repository. Returns the path of
// the new file in the local Git repository.
func copyToLocalGitRepo(srcPath, repo, gitPath string) (string, error) {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return "", err
	}
	defer srcFile.Close()

	var gitManifestPath string
	// If git path is not provided then use the name of the local file.
	if len(gitPath) == 0 {
		_, file := filepath.Split(srcPath)
		gitManifestPath = filepath.Join(repo, file)
	} else {
		gitManifestPath = filepath.Join(repo, gitPath)
	}

	// Create any directories in the local git repo path if necessary.
	if err := os.MkdirAll(filepath.Dir(gitManifestPath), os.ModePerm); err != nil {
		return "", err
	}

	dstFile, err := os.Create(gitManifestPath)
	if err != nil {
		return "", err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return "", err
	}
	return gitManifestPath, nil
}

// pollSyncStatus polls the sync status of the Argo application until it's synced or the timeout is reached.
func pollSyncStatus(name string, ns string, rev string, timeout time.Duration) error {
	ticker := time.NewTicker(argoSyncInterval)
	defer ticker.Stop()
	done := make(chan bool)
	go func() {
		time.Sleep(timeout)
		done <- true
	}()
	for {
		select {
		case <-done:
			return errors.New("timed out checking sync status of application")
		case <-ticker.C:
			fmt.Println("Tick...Checking the sync status")
			if err := checkSyncStatus(name, ns, rev); err != nil {
				fmt.Printf("%v\n", err)
				continue
			}
			return nil
		}
	}
}

// checkSyncStatus checks whether the Argo application is synced.
func checkSyncStatus(name string, ns string, headRev string) error {
	syncRev, err := queryPath(argoCRType, name, ns, "{.status.sync.revision}")
	if err != nil {
		return fmt.Errorf("error getting the application synced revision: %v", err)
	}

	if string(syncRev) != headRev {
		return fmt.Errorf("synced revision: %s does not match repository revision: %s", syncRev, headRev)
	}
	currentSyncStatus, err := queryPath(argoCRType, name, ns, "{.status.sync.status}")
	if err != nil {
		return fmt.Errorf("error getting the application synced status: %v", err)
	}

	if string(currentSyncStatus) != argoSyncedStatus {
		return fmt.Errorf("synced status does not match, status got: %s want: %s", string(currentSyncStatus), argoSyncedStatus)
	}
	return nil
}
