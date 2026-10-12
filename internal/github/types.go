package github

import "time"

// PullRequest holds GitHub PR metadata and discussions.
type PullRequest struct {
	Number   int
	Title    string
	Body     string
	Author   string
	MergedAt time.Time
	URL      string
	Comments []string // Review comments / debates
}

// Issue represents an issue or ticket linked to code changes.
type Issue struct {
	Number int
	Title  string
	State  string // "open" or "closed"
	URL    string
}
