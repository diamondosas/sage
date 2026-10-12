package graph

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	badger "github.com/dgraph-io/badger/v4"
	"sage/internal/ai"
	"sage/internal/git"
	"sage/internal/github"
)

// Stored records
type StoredPerson struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type StoredCommit struct {
	SHA       string    `json:"sha"`
	Message   string    `json:"message"`
	Author    string    `json:"author"`
	Timestamp time.Time `json:"timestamp"`
	Files     []string  `json:"files"`
}

type StoredExpertise struct {
	AuthorEmail string  `json:"email"`
	AuthorName  string  `json:"name"`
	Path        string  `json:"path"`
	Score       float64 `json:"score"`
	CommitCount int     `json:"commits"`
}

type StoredDecision struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Rationale string `json:"rationale"`
	PRNumber  int    `json:"pr_number"`
}

type StoredPR struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	URL    string `json:"url"`
	Author string `json:"author"`
}

type StoredOwnership struct {
	Pattern string   `json:"pattern"`
	Owners  []string `json:"owners"`
}

// Engine manages direct embedded storage with BadgerDB.
type Engine struct {
	db *badger.DB
	mu sync.RWMutex
}

// NewEngine initializes an embedded BadgerDB instance at dbPath.
// Defaults to filepath.Join(".github", "sage").
func NewEngine(dbPath string) (*Engine, error) {
	if dbPath == "" {
		dbPath = filepath.Join(".github", "sage")
	}

	opts := badger.DefaultOptions(dbPath)
	opts.Logger = nil // suppress verbose badger internal logging

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open Badger database at %s: %w", dbPath, err)
	}

	return &Engine{db: db}, nil
}

// Close closes the Badger database.
func (e *Engine) Close() error {
	if e != nil && e.db != nil {
		return e.db.Close()
	}
	return nil
}

// InitSchema is a no-op for Badger key-value engine.
func (e *Engine) InitSchema() error {
	return nil
}

// IngestGitHistory stores persons, commits, touched file indexes, and expertise.
func (e *Engine) IngestGitHistory(commits []git.Commit, expertise map[string][]git.ContributorExpertise) error {
	return e.db.Update(func(txn *badger.Txn) error {
		for _, c := range commits {
			// 1. Person
			personKey := fmt.Sprintf("person:%s", strings.ToLower(c.AuthorEmail))
			p := StoredPerson{Email: c.AuthorEmail, Name: c.AuthorName}
			if pBytes, err := json.Marshal(p); err == nil {
				_ = txn.Set([]byte(personKey), pBytes)
			}

			// 2. Commit
			commitKey := fmt.Sprintf("commit:%s", c.SHA)
			sc := StoredCommit{
				SHA:       c.SHA,
				Message:   c.Message,
				Author:    c.AuthorEmail,
				Timestamp: c.Timestamp,
				Files:     c.Files,
			}
			if cBytes, err := json.Marshal(sc); err == nil {
				_ = txn.Set([]byte(commitKey), cBytes)
			}

			// 3. Index: file -> commit
			for _, f := range c.Files {
				if strings.TrimSpace(f) == "" {
					continue
				}
				idxKey := fmt.Sprintf("file_commit:%s:%s", f, c.SHA)
				_ = txn.Set([]byte(idxKey), []byte(c.SHA))

				// Record file existence
				fileKey := fmt.Sprintf("file:%s", f)
				_ = txn.Set([]byte(fileKey), []byte(f))
			}
		}

		// 4. Ingest calculated contributor expertise
		for file, contribs := range expertise {
			for _, contrib := range contribs {
				exp := StoredExpertise{
					AuthorEmail: contrib.AuthorEmail,
					AuthorName:  contrib.AuthorName,
					Path:        file,
					Score:       contrib.Score,
					CommitCount: contrib.CommitCount,
				}
				if expBytes, err := json.Marshal(exp); err == nil {
					expKey := fmt.Sprintf("expertise:%s:%s", file, strings.ToLower(contrib.AuthorEmail))
					_ = txn.Set([]byte(expKey), expBytes)
				}
			}
		}

		return nil
	})
}

// IngestCodeowners stores ownership rules.
func (e *Engine) IngestCodeowners(rules []git.OwnershipRule) error {
	return e.db.Update(func(txn *badger.Txn) error {
		for _, r := range rules {
			stored := StoredOwnership{
				Pattern: r.Pattern,
				Owners:  r.Owners,
			}
			data, err := json.Marshal(stored)
			if err != nil {
				continue
			}
			key := fmt.Sprintf("ownership:%s", r.Pattern)
			if err := txn.Set([]byte(key), data); err != nil {
				return err
			}
		}
		return nil
	})
}

// IngestDecisions stores PRs and decisions and indexes them to touched files.
func (e *Engine) IngestDecisions(prs []github.PullRequest, decisions []ai.Decision) error {
	return e.db.Update(func(txn *badger.Txn) error {
		for i, pr := range prs {
			prKey := fmt.Sprintf("pr:%d", pr.Number)
			storedPR := StoredPR{
				Number: pr.Number,
				Title:  pr.Title,
				URL:    pr.URL,
				Author: pr.Author,
			}
			if prBytes, err := json.Marshal(storedPR); err == nil {
				_ = txn.Set([]byte(prKey), prBytes)
			}

			if i < len(decisions) {
				dec := decisions[i]
				storedDec := StoredDecision{
					ID:        dec.ID,
					Title:     dec.Title,
					Rationale: dec.Rationale,
					PRNumber:  pr.Number,
				}
				if decBytes, err := json.Marshal(storedDec); err == nil {
					decKey := fmt.Sprintf("decision:%s", dec.ID)
					_ = txn.Set([]byte(decKey), decBytes)

					prDecKey := fmt.Sprintf("pr_decision:%d:%s", pr.Number, dec.ID)
					_ = txn.Set([]byte(prDecKey), []byte(dec.ID))
				}
			}
		}
		return nil
	})
}

// QueryOwner finds formal owner and top 3 recent experts for a path.
func (e *Engine) QueryOwner(path string) (string, []string, error) {
	formalOwner := "Unassigned (Check CODEOWNERS)"
	var experts []string

	err := e.db.View(func(txn *badger.Txn) error {
		// 1. Look for matching ownership rule
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		prefix := []byte("ownership:")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			_ = item.Value(func(val []byte) error {
				var rule StoredOwnership
				if err := json.Unmarshal(val, &rule); err == nil {
					cleanPattern := strings.TrimPrefix(rule.Pattern, "/")
					cleanPattern = strings.TrimSuffix(cleanPattern, "*")
					cleanPattern = strings.TrimSuffix(cleanPattern, "/")
					cleanPath := strings.TrimPrefix(path, "/")

					if cleanPattern == "" || strings.HasPrefix(cleanPath, cleanPattern) || cleanPath == cleanPattern {
						if len(rule.Owners) > 0 {
							formalOwner = strings.Join(rule.Owners, ", ")
						}
					}
				}
				return nil
			})
		}

		// 2. Aggregate expertise scores for the path
		scoreMap := make(map[string]*StoredExpertise)

		expIt := txn.NewIterator(badger.DefaultIteratorOptions)
		defer expIt.Close()

		expPrefix := []byte("expertise:")
		cleanPath := strings.TrimPrefix(path, "/")
		for expIt.Seek(expPrefix); expIt.ValidForPrefix(expPrefix); expIt.Next() {
			item := expIt.Item()
			_ = item.Value(func(val []byte) error {
				var exp StoredExpertise
				if err := json.Unmarshal(val, &exp); err == nil {
					expClean := strings.TrimPrefix(exp.Path, "/")
					if expClean == cleanPath || strings.HasPrefix(expClean, cleanPath) || strings.HasPrefix(cleanPath, expClean) {
						key := exp.AuthorEmail
						if existing, ok := scoreMap[key]; ok {
							existing.Score += exp.Score
							existing.CommitCount += exp.CommitCount
						} else {
							scoreCopy := exp
							scoreMap[key] = &scoreCopy
						}
					}
				}
				return nil
			})
		}

		var list []*StoredExpertise
		for _, exp := range scoreMap {
			list = append(list, exp)
		}
		sort.Slice(list, func(i, j int) bool {
			return list[i].Score > list[j].Score
		})

		for i := 0; i < len(list) && i < 3; i++ {
			exp := list[i]
			name := exp.AuthorName
			if name == "" {
				name = exp.AuthorEmail
			}
			experts = append(experts, fmt.Sprintf("%s (%s) — Activity Score: %.2f", name, exp.AuthorEmail, exp.Score))
		}

		return nil
	})

	return formalOwner, experts, err
}

// QueryWhy traces the origin and decision lineage for a file or commit.
func (e *Engine) QueryWhy(target string) ([]string, error) {
	var results []string

	err := e.db.View(func(txn *badger.Txn) error {
		// Collect matching commits
		var matchedCommits []StoredCommit

		// Check if target is directly a commit SHA
		commitKeyPrefix := []byte("commit:")
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		for it.Seek(commitKeyPrefix); it.ValidForPrefix(commitKeyPrefix); it.Next() {
			item := it.Item()
			_ = item.Value(func(val []byte) error {
				var sc StoredCommit
				if err := json.Unmarshal(val, &sc); err == nil {
					// Check target match
					isMatch := false
					if strings.HasPrefix(sc.SHA, target) {
						isMatch = true
					} else {
						for _, f := range sc.Files {
							if strings.Contains(f, target) || f == target {
								isMatch = true
								break
							}
						}
					}
					if isMatch {
						matchedCommits = append(matchedCommits, sc)
					}
				}
				return nil
			})
		}

		// Sort commits by timestamp descending
		sort.Slice(matchedCommits, func(i, j int) bool {
			return matchedCommits[i].Timestamp.After(matchedCommits[j].Timestamp)
		})

		// Load decisions
		var decisions []StoredDecision
		decIt := txn.NewIterator(badger.DefaultIteratorOptions)
		defer decIt.Close()
		decPrefix := []byte("decision:")
		for decIt.Seek(decPrefix); decIt.ValidForPrefix(decPrefix); decIt.Next() {
			item := decIt.Item()
			_ = item.Value(func(val []byte) error {
				var dec StoredDecision
				if err := json.Unmarshal(val, &dec); err == nil {
					decisions = append(decisions, dec)
				}
				return nil
			})
		}

		for i, sc := range matchedCommits {
			if i >= 5 {
				break
			}
			sha := sc.SHA
			if len(sha) > 7 {
				sha = sha[:7]
			}
			entry := fmt.Sprintf("Commit [%s]: %s", sha, sc.Message)

			// Try to match decisions linked to this commit via PR mentions in commit message
			for _, dec := range decisions {
				prTag := fmt.Sprintf("#%d", dec.PRNumber)
				if strings.Contains(sc.Message, prTag) {
					if dec.Title != "" {
						entry += fmt.Sprintf("\n  Decision: %s\n  Rationale: %s", dec.Title, dec.Rationale)
					}
				}
			}

			results = append(results, entry)
		}

		return nil
	})

	return results, err
}

// QueryOverview returns high-level storage statistics without external containers.
func (e *Engine) QueryOverview() (map[string]int, error) {
	stats := map[string]int{
		"Files":     0,
		"Commits":   0,
		"People":    0,
		"Decisions": 0,
	}

	err := e.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.IteratorOptions{PrefetchValues: false})
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			key := string(it.Item().Key())
			switch {
			case strings.HasPrefix(key, "file:"):
				stats["Files"]++
			case strings.HasPrefix(key, "commit:"):
				stats["Commits"]++
			case strings.HasPrefix(key, "person:"):
				stats["People"]++
			case strings.HasPrefix(key, "decision:"):
				stats["Decisions"]++
			}
		}
		return nil
	})

	return stats, err
}

// ExecuteRawCypher executes raw key-value inspection queries in Badger.
func (e *Engine) ExecuteRawCypher(query string) ([][]string, error) {
	// For compatibility with `sage query <arg>`, support key prefix scans or raw keys
	var rows [][]string
	trimmed := strings.TrimSpace(query)

	err := e.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		prefix := []byte(trimmed)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			k := string(item.Key())
			_ = item.Value(func(val []byte) error {
				rows = append(rows, []string{k, string(val)})
				return nil
			})
			if len(rows) >= 50 {
				break
			}
		}
		return nil
	})

	return rows, err
}
