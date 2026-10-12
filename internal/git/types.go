package git

import "time"

// Commit represents a single Git commit record.
type Commit struct {
	SHA         string
	AuthorName  string
	AuthorEmail string
	Message     string
	Timestamp   time.Time
	Files       []string // Files modified in this commit
}

// OwnershipRule represents a line parsed from .github/CODEOWNERS.
type OwnershipRule struct {
	Pattern string   // Path or pattern (e.g. "/internal/auth/*")
	Owners  []string // GitHub handles or emails (e.g. "@security-team")
}

// ContributorExpertise represents calculated tribal knowledge for a file/directory.
type ContributorExpertise struct {
	AuthorName string
	AuthorEmail string
	Score       float64 // Recency-weighted score (higher = more recent/frequent changes)
	CommitCount int
}
