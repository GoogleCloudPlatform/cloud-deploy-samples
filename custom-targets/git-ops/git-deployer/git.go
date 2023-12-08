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
)

const (
	gitBin = "git"
	remote = "origin"
)

// gitRepository holds the repository values for git commands.
type gitRepository struct {
	dir      string
	hostname string
	owner    string
	repoName string
	email    string
	username string
}

// newGitRepository returns a gitRepository to interact with a repository.
func newGitRepository(hostname, owner, repoName, email, username string) *gitRepository {
	return &gitRepository{
		hostname: hostname,
		owner:    owner,
		repoName: repoName,
		email:    email,
		username: username,
	}
}

// cloneRepo clones a Git repository to the local filesystem.
func (g *gitRepository) cloneRepo(secret string) ([]byte, error) {
	args := []string{"clone", fmt.Sprintf("https://%s:%s@%s/%s/%s.git", g.owner, secret, g.hostname, g.owner, g.repoName)}
	g.dir = g.repoName
	return runCmd(gitBin, args, "", false)
}

// checkoutBranch checkouts and resets an existing branch or creates a new one.
func (g *gitRepository) checkoutBranch(branch string) ([]byte, error) {
	args := []string{"checkout", "-B", branch}
	return runCmd(gitBin, args, g.dir, true)
}

// add adds all the files in the working tree to the index.
func (g *gitRepository) add() ([]byte, error) {
	args := []string{"add", "."}
	return runCmd(gitBin, args, g.dir, true)
}

// detectDiff gets the working tree status and uses the porcelain command to simplify scripting.
func (g *gitRepository) detectDiff() ([]byte, error) {
	args := []string{"status", "--porcelain"}
	return runCmd(gitBin, args, g.dir, true)
}

// commit commits the changes in the index to the repository.
// It supports configuring the username, email and message for the commit.
func (g *gitRepository) commit(msg string) ([]byte, error) {
	args := []string{"-c", fmt.Sprintf("user.email=%s", g.email), "-c", fmt.Sprintf("user.name=%s", g.username), "commit", "-a", "-m", msg}
	return runCmd(gitBin, args, g.dir, true)
}

// push pushes the changes a remote branch.
func (g *gitRepository) push(branch string) ([]byte, error) {
	args := []string{"push", remote, branch}
	return runCmd(gitBin, args, g.dir, true)
}

// checkIfExists checks if a branch exists on the remote.
func (g *gitRepository) checkIfExists(branch string) ([]byte, error) {
	args := []string{"ls-remote", "--heads", remote, fmt.Sprintf("refs/heads/%s", branch)}
	return runCmd(gitBin, args, g.dir, true)
}

// pull pulls changes from a remote branch.
func (g *gitRepository) pull(branch string) ([]byte, error) {
	args := []string{"pull", remote, branch}
	return runCmd(gitBin, args, g.dir, true)
}
