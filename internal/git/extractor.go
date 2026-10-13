package git

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Extractor extracts commits, changed files, and CODEOWNERS rules from Git.
type Extractor struct {
	RepoPath      string
	MaxCommits    int
	DecayHalfLife time.Duration
}

// NewExtractor creates a new Git extractor instance.
func NewExtractor(repoPath string, maxCommits int, decayHalfLife time.Duration) *Extractor {
	return &Extractor{
		RepoPath:      repoPath,
		MaxCommits:    maxCommits,
		DecayHalfLife: decayHalfLife,
	}
}

// GetHeadSHA returns the current Git HEAD commit SHA.
func (e *Extractor) GetHeadSHA() (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = e.RepoPath
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// ExtractCommits reads recent commit history via standard git log.
func (e *Extractor) ExtractCommits() ([]Commit, error) {
	return e.runGitLog()
}

// ExtractCommitsSince extracts commits between sinceSHA and HEAD incrementally.
// If sinceSHA is empty or invalid, falls back to ExtractCommits.
func (e *Extractor) ExtractCommitsSince(sinceSHA string) ([]Commit, error) {
	if sinceSHA == "" {
		return e.ExtractCommits()
	}

	// Verify commit exists in repository history
	verifyCmd := exec.Command("git", "cat-file", "-e", sinceSHA)
	verifyCmd.Dir = e.RepoPath
	if err := verifyCmd.Run(); err != nil {
		// Commit not found (shallow clone or rebased history) -> full extraction
		return e.ExtractCommits()
	}

	return e.runGitLog(fmt.Sprintf("%s..HEAD", sinceSHA))
}

func (e *Extractor) runGitLog(extraArgs ...string) ([]Commit, error) {
	args := []string{
		"log",
		"--name-only",
		"--pretty=format:COMMIT_START%n%H|%an|%ae|%aI|%s",
	}
	args = append(args, extraArgs...)
	if e.MaxCommits > 0 {
		args = append(args, "-n", strconv.Itoa(e.MaxCommits))
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = e.RepoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		outStr := string(output)
		// Return empty list gracefully if repository has no commits yet
		if strings.Contains(outStr, "does not have any commits yet") {
			return nil, nil
		}
		return nil, fmt.Errorf("git log error: %w (%s)", err, strings.TrimSpace(outStr))
	}

	return parseGitLogOutput(output)
}

// GetChangedFilesSince detects files added, modified, or deleted since sinceSHA.
// If sinceSHA is empty or invalid, returns all tracked files via git ls-files.
func (e *Extractor) GetChangedFilesSince(sinceSHA string) ([]string, []string, error) {
	if sinceSHA == "" {
		files, err := e.ListTrackedFiles()
		return files, nil, err
	}

	// Verify commit exists
	verifyCmd := exec.Command("git", "cat-file", "-e", sinceSHA)
	verifyCmd.Dir = e.RepoPath
	if err := verifyCmd.Run(); err != nil {
		files, err := e.ListTrackedFiles()
		return files, nil, err
	}

	cmd := exec.Command("git", "diff", "--name-status", fmt.Sprintf("%s..HEAD", sinceSHA))
	cmd.Dir = e.RepoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, nil, fmt.Errorf("git diff error: %w", err)
	}

	var addedOrModified []string
	var deleted []string
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) >= 2 {
			status := strings.TrimSpace(parts[0])
			path := strings.TrimSpace(parts[len(parts)-1]) // Support rename syntax R100 old new
			if strings.HasPrefix(status, "D") {
				deleted = append(deleted, path)
			} else {
				addedOrModified = append(addedOrModified, path)
			}
		}
	}
	return addedOrModified, deleted, nil
}

// ListTrackedFiles lists all tracked files in the repository.
func (e *Extractor) ListTrackedFiles() ([]string, error) {
	cmd := exec.Command("git", "ls-files")
	cmd.Dir = e.RepoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git ls-files error: %w", err)
	}

	var files []string
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

func parseGitLogOutput(output []byte) ([]Commit, error) {
	var commits []Commit
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	var current *Commit
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if line == "COMMIT_START" {
			if current != nil {
				commits = append(commits, *current)
			}
			current = &Commit{}
			continue
		}

		if current != nil && current.SHA == "" {
			parts := strings.SplitN(line, "|", 5)
			if len(parts) >= 5 {
				current.SHA = parts[0]
				current.AuthorName = parts[1]
				current.AuthorEmail = parts[2]
				if t, err := time.Parse(time.RFC3339, parts[3]); err == nil {
					current.Timestamp = t
				}
				current.Message = parts[4]
			}
			continue
		}

		// Lines following header are file paths
		if current != nil && current.SHA != "" {
			current.Files = append(current.Files, line)
		}
	}

	if current != nil && current.SHA != "" {
		commits = append(commits, *current)
	}

	return commits, nil
}

// CalculateExpertise computes recency-weighted activity scores per file.
func (e *Extractor) CalculateExpertise(commits []Commit) map[string][]ContributorExpertise {
	now := time.Now()
	// Map: FilePath -> AuthorEmail -> Raw Score
	scores := make(map[string]map[string]*ContributorExpertise)

	halfLifeSeconds := e.DecayHalfLife.Seconds()
	if halfLifeSeconds <= 0 {
		halfLifeSeconds = (60 * 24 * time.Hour).Seconds()
	}

	for _, c := range commits {
		ageSeconds := now.Sub(c.Timestamp).Seconds()
		if ageSeconds < 0 {
			ageSeconds = 0
		}

		// Exponential decay: Score = 2 ^ (-age / halfLife)
		// CHANGE HERE: Change decay function (e.g. linear or step) if desired
		weight := math.Pow(2, -ageSeconds/halfLifeSeconds)

		for _, file := range c.Files {
			if _, ok := scores[file]; !ok {
				scores[file] = make(map[string]*ContributorExpertise)
			}
			contrib, ok := scores[file][c.AuthorEmail]
			if !ok {
				contrib = &ContributorExpertise{
					AuthorEmail: c.AuthorEmail,
					Score:       0,
					CommitCount: 0,
				}
				scores[file][c.AuthorEmail] = contrib
			}
			contrib.Score += weight
			contrib.CommitCount++
		}
	}

	result := make(map[string][]ContributorExpertise)
	for file, contribMap := range scores {
		var list []ContributorExpertise
		for _, item := range contribMap {
			list = append(list, *item)
		}
		result[file] = list
	}

	return result
}

// ParseCodeowners looks for CODEOWNERS in standard locations (.github/, root, docs/).
func (e *Extractor) ParseCodeowners() ([]OwnershipRule, error) {
	locations := []string{
		filepath.Join(e.RepoPath, ".github", "CODEOWNERS"),
		filepath.Join(e.RepoPath, "CODEOWNERS"),
		filepath.Join(e.RepoPath, "docs", "CODEOWNERS"),
	}

	for _, path := range locations {
		if _, err := os.Stat(path); err == nil {
			return parseCodeownersFile(path)
		}
	}

	return nil, nil // No CODEOWNERS file present
}

func parseCodeownersFile(path string) ([]OwnershipRule, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var rules []OwnershipRule
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Ignore empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 2 {
			rules = append(rules, OwnershipRule{
				Pattern: fields[0],
				Owners:  fields[1:],
			})
		}
	}

	return rules, scanner.Err()
}
