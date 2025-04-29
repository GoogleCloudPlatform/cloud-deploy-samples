package providers

import (
	"fmt"
	"time"
)

// GitProvider interface provides methods for interacting with the API of a Git Provider.
type GitProvider interface {
	OpenPullRequest(src, dst, title, body string) (*PullRequest, error)
	MergePullRequest(prNo int) (*MergeResponse, error)
}

// PullRequest represents a pull request resource from a Git provider.
type PullRequest struct {
	Number int
}

// MergeResponse represents the response from a Git provider when merging a pull request.
type MergeResponse struct {
	Sha string
}

// CreateProvider returns an instance of the GitProvider. Returns an error if an unsupported
// provider hostname is provided.
func CreateProvider(hostname, repoName, owner, secret string) (GitProvider, error) {
	var provider GitProvider
	switch hostname {
	case "github.com":
		provider = &GitHubProvider{
			Repository: repoName,
			Token:      secret,
			Owner:      owner,
		}
	case "gitlab.com":
		provider = &GitLabProvider{
			Repository: repoName,
			Token:      secret,
			Owner:      owner,
		}
	default:
		return nil, fmt.Errorf("unsupported git provider: %s", hostname)
	}
	return provider, nil
}

func mergePullRequestWithRetries(prNo int, call func(prNo int) (*MergeResponse, error)) (*MergeResponse, error) {
	endTime := time.Now().Add(2 * time.Minute)
	startWait := time.Second * 2
	var mr *MergeResponse
	var err error
	for attempts := 1; time.Now().Before(endTime); attempts++ {
		mr, err = call(prNo)
		if err != nil {
			time.Sleep(startWait * time.Duration(attempts))
			continue
		}
		return mr, err
	}
	return nil, err
}
