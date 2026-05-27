package fetcher

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/go-github/v62/github"
)

const rawGithubBase = "https://raw.githubusercontent.com"

// FrameworkFile represents a fetched framework YAML file.
type FrameworkFile struct {
	RefID        string
	Jurisdiction string
	Filename     string
	Content      []byte
}

// GitHubFetcher downloads CISO Assistant framework YAMLs from GitHub.
type GitHubFetcher struct {
	owner      string
	repo       string
	path       string
	client     *github.Client
	httpClient *http.Client
	cacheDir   string
}

// NewGitHubFetcher creates a fetcher for the given GitHub repository path.
// If token is empty, unauthenticated requests are used (lower rate limits).
func NewGitHubFetcher(owner, repo, path string) *GitHubFetcher {
	token := os.Getenv("GITHUB_TOKEN")
	var client *github.Client
	if token != "" {
		client = github.NewClient(nil).WithAuthToken(token)
	} else {
		client = github.NewClient(nil)
	}

	cacheDir := filepath.Join("data", "cache")
	_ = os.MkdirAll(cacheDir, 0755)

	return &GitHubFetcher{
		owner:      owner,
		repo:       repo,
		path:       path,
		client:     client,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		cacheDir:   cacheDir,
	}
}

// Fetch lists YAML files in the target directory and downloads those matching the filter.
// filter is a comma-separated list of ref_id prefixes or glob patterns (e.g. "cmmc-2.0,nist-csf-2.0,bs-it-gs-2023-*").
func (f *GitHubFetcher) Fetch(ctx context.Context, filter string) ([]FrameworkFile, error) {
	patterns := parseFilter(filter)

	// List files via GitHub Tree API (recursive depth 1 for the target path).
	tree, _, err := f.client.Git.GetTree(ctx, f.owner, f.repo, "main", true)
	if err != nil {
		// Fallback: try listing directory contents via Repository Content API.
		return f.fetchViaContentsAPI(ctx, patterns)
	}

	var files []FrameworkFile
	for _, entry := range tree.Entries {
		if entry.GetType() != "blob" {
			continue
		}
		path := entry.GetPath()
		if !strings.HasPrefix(path, f.path) {
			continue
		}
		if !strings.HasSuffix(path, ".yaml") {
			continue
		}
		refID := strings.TrimSuffix(filepath.Base(path), ".yaml")
		if !matchFilter(refID, patterns) {
			continue
		}
		content, err := f.download(ctx, path, entry.GetSHA())
		if err != nil {
			return nil, fmt.Errorf("download %s: %w", path, err)
		}
		files = append(files, FrameworkFile{
			RefID:    refID,
			Filename: filepath.Base(path),
			Content:  content,
		})
	}
	return files, nil
}

func (f *GitHubFetcher) fetchViaContentsAPI(ctx context.Context, patterns []string) ([]FrameworkFile, error) {
	_, dirContents, _, err := f.client.Repositories.GetContents(ctx, f.owner, f.repo, f.path, nil)
	if err != nil {
		return nil, fmt.Errorf("list contents %s/%s/%s: %w", f.owner, f.repo, f.path, err)
	}

	var files []FrameworkFile
	for _, item := range dirContents {
		if item.GetType() != "file" {
			continue
		}
		if !strings.HasSuffix(item.GetName(), ".yaml") {
			continue
		}
		refID := strings.TrimSuffix(item.GetName(), ".yaml")
		if !matchFilter(refID, patterns) {
			continue
		}
		content, err := f.download(ctx, filepath.Join(f.path, item.GetName()), item.GetSHA())
		if err != nil {
			return nil, fmt.Errorf("download %s: %w", item.GetName(), err)
		}
		files = append(files, FrameworkFile{
			RefID:    refID,
			Filename: item.GetName(),
			Content:  content,
		})
	}
	return files, nil
}

// download fetches the raw blob via raw.githubusercontent.com with local file caching.
// Retries on transient network errors and 5xx responses with exponential backoff.
func (f *GitHubFetcher) download(ctx context.Context, repoPath, sha string) ([]byte, error) {
	cacheFile := filepath.Join(f.cacheDir, fmt.Sprintf("%s_%s.yaml", strings.ReplaceAll(repoPath, "/", "_"), sha[:7]))
	if cached, err := os.ReadFile(cacheFile); err == nil {
		return cached, nil
	}

	url := fmt.Sprintf("%s/%s/%s/main/%s", rawGithubBase, f.owner, f.repo, repoPath)
	var data []byte
	const maxRetries = 3
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(attempt) * time.Second):
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		if token := os.Getenv("GITHUB_TOKEN"); token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}

		resp, err := f.httpClient.Do(req)
		if err != nil {
			if attempt == maxRetries {
				return nil, fmt.Errorf("download %s: %w", repoPath, err)
			}
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			if attempt == maxRetries {
				return nil, fmt.Errorf("read body %s: %w", repoPath, err)
			}
			continue
		}
		if resp.StatusCode == http.StatusOK {
			data = body
			break
		}
		if resp.StatusCode >= 500 && attempt < maxRetries {
			continue // retry on server errors
		}
		return nil, fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}
	_ = os.WriteFile(cacheFile, data, 0644)
	return data, nil
}

func parseFilter(filter string) []string {
	if filter == "" {
		return nil
	}
	parts := strings.Split(filter, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func matchFilter(refID string, patterns []string) bool {
	if len(patterns) == 0 {
		return true
	}
	for _, p := range patterns {
		if p == refID {
			return true
		}
		if strings.HasSuffix(p, "*") {
			prefix := strings.TrimSuffix(p, "*")
			if strings.HasPrefix(refID, prefix) {
				return true
			}
		}
		// Try regex for more complex patterns.
		re, err := regexp.Compile("^" + strings.ReplaceAll(p, "*", ".*") + "$")
		if err == nil && re.MatchString(refID) {
			return true
		}
	}
	return false
}
