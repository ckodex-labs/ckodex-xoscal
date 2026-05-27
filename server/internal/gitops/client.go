package gitops

import (
	"context"
	"fmt"
	"os"

	"github.com/google/go-github/v62/github"
)

// Client wraps go-github for OSCAL artifact lifecycle operations.
type Client struct {
	gh *github.Client
}

// NewClient creates a GitOps client using GITHUB_TOKEN or unauthenticated access.
func NewClient() *Client {
	token := os.Getenv("GITHUB_TOKEN")
	var client *github.Client
	if token != "" {
		client = github.NewClient(nil).WithAuthToken(token)
	} else {
		client = github.NewClient(nil)
	}
	return &Client{gh: client}
}

// PublishRelease creates a GitHub release with the given tag and artifacts.
func (c *Client) PublishRelease(ctx context.Context, owner, repo, tag, target, name, body string, assetPaths []string) (string, string, error) {
	release, _, err := c.gh.Repositories.CreateRelease(ctx, owner, repo, &github.RepositoryRelease{
		TagName:         github.String(tag),
		TargetCommitish: github.String(target),
		Name:            github.String(name),
		Body:            github.String(body),
	})
	if err != nil {
		return "", "", fmt.Errorf("create release: %w", err)
	}
	return release.GetHTMLURL(), release.GetUploadURL(), nil
}

// ProposeMappingUpdate opens a pull request for mapping changes.
func (c *Client) ProposeMappingUpdate(ctx context.Context, owner, repo, base, head, title, body string) (string, int, error) {
	pr, _, err := c.gh.PullRequests.Create(ctx, owner, repo, &github.NewPullRequest{
		Title: github.String(title),
		Head:  github.String(head),
		Base:  github.String(base),
		Body:  github.String(body),
	})
	if err != nil {
		return "", 0, fmt.Errorf("create PR: %w", err)
	}
	return pr.GetHTMLURL(), pr.GetNumber(), nil
}
