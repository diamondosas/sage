package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Client handles GitHub REST API interactions.
type Client struct {
	Token      string
	HTTPClient *http.Client
}

// NewClient initializes a new GitHub client.
func NewClient(token string) *Client {
	return &Client{
		Token: token,
		HTTPClient: &http.Client{
			// CHANGE HERE: Modify API timeout if running on slow networks
			Timeout: 15 * time.Second,
		},
	}
}

// DetectRepoSlug detects "owner/repo" from local git remote origin.
func DetectRepoSlug(repoDir string) (string, error) {
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	url := strings.TrimSpace(string(out))
	// Handles git@github.com:owner/repo.git or https://github.com/owner/repo.git
	re := regexp.MustCompile(`github\.com[:/]([^/]+)/([^/.]+?)(\.git)?$`)
	matches := re.FindStringSubmatch(url)
	if len(matches) >= 3 {
		return fmt.Sprintf("%s/%s", matches[1], matches[2]), nil
	}
	return "", fmt.Errorf("unable to detect GitHub repository slug from remote: %s", url)
}

// FetchMergedPRs fetches recently merged pull requests.
func (c *Client) FetchMergedPRs(repoSlug string, limit int) ([]PullRequest, error) {
	// GitHub REST API: GET /repos/{owner}/{repo}/pulls?state=closed&per_page=...
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/pulls?state=closed&per_page=%d", repoSlug, limit)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status: %d", resp.StatusCode)
	}

	var rawPRs []struct {
		Number   int        `json:"number"`
		Title    string     `json:"title"`
		Body     string     `json:"body"`
		HTMLURL  string     `json:"html_url"`
		MergedAt *time.Time `json:"merged_at"`
		User     struct {
			Login string `json:"login"`
		} `json:"user"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rawPRs); err != nil {
		return nil, err
	}

	var prs []PullRequest
	for _, raw := range rawPRs {
		// Only include PRs that were actually merged (not simply closed without merge)
		if raw.MergedAt == nil {
			continue
		}
		pr := PullRequest{
			Number:   raw.Number,
			Title:    raw.Title,
			Body:     raw.Body,
			Author:   raw.User.Login,
			MergedAt: *raw.MergedAt,
			URL:      raw.HTMLURL,
		}
		prs = append(prs, pr)
	}

	return prs, nil
}

// ExtractLinkedIssues extracts issue numbers from PR body (e.g., "Fixes #12", "Closes #45").
func ExtractLinkedIssues(prBody string) []int {
	re := regexp.MustCompile(`(?i)(?:fixes|closes|resolves)\s+#(\d+)`)
	matches := re.FindAllStringSubmatch(prBody, -1)

	var issues []int
	seen := make(map[int]bool)
	for _, m := range matches {
		if len(m) >= 2 {
			if num, err := strconv.Atoi(m[1]); err == nil && !seen[num] {
				issues = append(issues, num)
				seen[num] = true
			}
		}
	}
	return issues
}
