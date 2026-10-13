package graph

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"sage/internal/ai"
	"sage/internal/git"
	"sage/internal/github"
)

// StoredOwnership matches CODEOWNERS rule json storage.
type StoredOwnership struct {
	Pattern string   `json:"pattern"`
	Owners  []string `json:"owners"`
}

// Engine wraps an embedded single-file SQLite database.
type Engine struct {
	db   *sql.DB
	path string
}

// NewEngine initializes the single-file SQLite database at dbPath.
// Defaults to .github/sage/sage.db.
func NewEngine(dbPath string) (*Engine, error) {
	if dbPath == "" {
		dbPath = filepath.Join(".github", "sage")
	}

	var filePath string
	if strings.HasSuffix(dbPath, ".db") {
		filePath = dbPath
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	} else {
		if err := os.MkdirAll(dbPath, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dbPath, err)
		}
		filePath = filepath.Join(dbPath, "sage.db")
	}

	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on&_synchronous=NORMAL", filePath)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database at %s: %w", filePath, err)
	}

	// Optimize connection pooling for embedded SQLite
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	eng := &Engine{
		db:   db,
		path: filePath,
	}

	if err := eng.InitSchema(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to initialize SQLite schema: %w", err)
	}

	return eng, nil
}

// Close gracefully closes the SQLite database connection.
func (e *Engine) Close() error {
	if e == nil || e.db == nil {
		return nil
	}
	return e.db.Close()
}

// InitSchema creates tables and indexes if they do not exist.
func (e *Engine) InitSchema() error {
	_, err := e.db.Exec(SchemaDDL)
	return err
}

// Path returns the path of the underlying SQLite file.
func (e *Engine) Path() string {
	return e.path
}

// --- Sync Metadata Tracking ---

// GetGitLastSyncedCommit returns the SHA of the last commit synced to git history.
func (e *Engine) GetGitLastSyncedCommit() (string, error) {
	return e.getMeta("git_last_synced_commit")
}

// SetGitLastSyncedCommit records the latest commit SHA in git history.
func (e *Engine) SetGitLastSyncedCommit(sha string) error {
	return e.setMeta("git_last_synced_commit", sha)
}

// GetMindLastSyncedCommit returns the SHA of the last commit synced to AI Mind.
func (e *Engine) GetMindLastSyncedCommit() (string, error) {
	return e.getMeta("mind_last_synced_commit")
}

// SetMindLastSyncedCommit records the latest commit SHA in AI Mind.
func (e *Engine) SetMindLastSyncedCommit(sha string) error {
	return e.setMeta("mind_last_synced_commit", sha)
}

func (e *Engine) getMeta(key string) (string, error) {
	var val string
	err := e.db.QueryRow("SELECT val FROM meta WHERE key = ?", key).Scan(&val)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return val, err
}

func (e *Engine) setMeta(key, val string) error {
	_, err := e.db.Exec(`
		INSERT INTO meta (key, val) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET val = excluded.val
	`, key, val)
	return err
}

// --- Git Graph Operations ---

// IngestGitHistory stores persons, commits, touched files, and updates recency-decay expertise scores.
func (e *Engine) IngestGitHistory(commits []git.Commit, expertise map[string][]git.ContributorExpertise) error {
	tx, err := e.db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmtPerson, err := tx.Prepare(`
		INSERT INTO people (email, name) VALUES (?, ?)
		ON CONFLICT(email) DO UPDATE SET name = CASE WHEN excluded.name != '' THEN excluded.name ELSE people.name END
	`)
	if err != nil {
		return err
	}
	defer stmtPerson.Close()

	stmtCommit, err := tx.Prepare(`
		INSERT INTO commits (sha, author_email, message, timestamp) VALUES (?, ?, ?, ?)
		ON CONFLICT(sha) DO UPDATE SET message = excluded.message
	`)
	if err != nil {
		return err
	}
	defer stmtCommit.Close()

	stmtCommitFile, err := tx.Prepare(`
		INSERT OR IGNORE INTO commit_files (commit_sha, file_path) VALUES (?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmtCommitFile.Close()

	stmtFileRecord, err := tx.Prepare(`
		INSERT OR IGNORE INTO files (path, updated_at) VALUES (?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmtFileRecord.Close()

	now := time.Now()
	seenFiles := make(map[string]bool)

	for _, c := range commits {
		email := strings.ToLower(strings.TrimSpace(c.AuthorEmail))
		if email != "" {
			if _, err := stmtPerson.Exec(email, c.AuthorName); err != nil {
				return fmt.Errorf("insert person %s: %w", email, err)
			}
		}

		if _, err := stmtCommit.Exec(c.SHA, email, c.Message, c.Timestamp); err != nil {
			return fmt.Errorf("insert commit %s: %w", c.SHA, err)
		}

		for _, f := range c.Files {
			cleanF := strings.TrimSpace(f)
			if cleanF == "" {
				continue
			}
			if _, err := stmtCommitFile.Exec(c.SHA, cleanF); err != nil {
				return fmt.Errorf("insert commit_file %s -> %s: %w", c.SHA, cleanF, err)
			}
			if !seenFiles[cleanF] {
				seenFiles[cleanF] = true
				if _, err := stmtFileRecord.Exec(cleanF, now); err != nil {
					return fmt.Errorf("insert file %s: %w", cleanF, err)
				}
			}
		}
	}

	stmtExpertise, err := tx.Prepare(`
		INSERT INTO expertise (file_path, author_email, author_name, score, commit_count)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(file_path, author_email) DO UPDATE SET
			score = expertise.score + excluded.score,
			commit_count = expertise.commit_count + excluded.commit_count,
			author_name = CASE WHEN excluded.author_name != '' THEN excluded.author_name ELSE expertise.author_name END
	`)
	if err != nil {
		return err
	}
	defer stmtExpertise.Close()

	for file, contribs := range expertise {
		cleanFile := strings.TrimSpace(file)
		if cleanFile == "" {
			continue
		}
		for _, contrib := range contribs {
			email := strings.ToLower(strings.TrimSpace(contrib.AuthorEmail))
			if email == "" {
				continue
			}
			if _, err := stmtExpertise.Exec(cleanFile, email, contrib.AuthorName, contrib.Score, contrib.CommitCount); err != nil {
				return fmt.Errorf("upsert expertise %s -> %s: %w", cleanFile, email, err)
			}
		}
	}

	return tx.Commit()
}

// IngestCodeowners stores CODEOWNERS rules.
func (e *Engine) IngestCodeowners(rules []git.OwnershipRule) error {
	tx, err := e.db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(`
		INSERT INTO codeowners (pattern, owners) VALUES (?, ?)
		ON CONFLICT(pattern) DO UPDATE SET owners = excluded.owners
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, r := range rules {
		data, err := json.Marshal(r.Owners)
		if err != nil {
			continue
		}
		if _, err := stmt.Exec(r.Pattern, string(data)); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// --- AI Mind Operations ---

// GetStoredFileHash returns the stored content hash for a file.
func (e *Engine) GetStoredFileHash(path string) (string, error) {
	var hash sql.NullString
	err := e.db.QueryRow("SELECT hash FROM files WHERE path = ?", path).Scan(&hash)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return hash.String, nil
}

// IngestFileAnalysis stores AI file analyses and updates role/hash metadata.
func (e *Engine) IngestFileAnalysis(analyses []ai.FileAnalysis) error {
	tx, err := e.db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(`
		INSERT INTO files (path, summary, role, exports, dependencies, hash, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET
			summary = excluded.summary,
			role = excluded.role,
			exports = excluded.exports,
			dependencies = excluded.dependencies,
			hash = excluded.hash,
			updated_at = excluded.updated_at
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, a := range analyses {
		exportsJSON, _ := json.Marshal(a.Exports)
		depsJSON, _ := json.Marshal(a.Dependencies)
		updatedAt := a.UpdatedAt
		if updatedAt.IsZero() {
			updatedAt = time.Now()
		}

		if _, err := stmt.Exec(a.Path, a.Summary, a.Role, string(exportsJSON), string(depsJSON), a.Hash, updatedAt); err != nil {
			return fmt.Errorf("upsert file analysis %s: %w", a.Path, err)
		}
	}

	return tx.Commit()
}

// DeleteFileAnalysis removes file analyses for deleted files.
func (e *Engine) DeleteFileAnalysis(paths []string) error {
	if len(paths) == 0 {
		return nil
	}

	tx, err := e.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare("DELETE FROM files WHERE path = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, p := range paths {
		if _, err := stmt.Exec(p); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetFileAnalysis returns the stored AI analysis for a file path.
func (e *Engine) GetFileAnalysis(path string) (*ai.FileAnalysis, error) {
	var summary, role, exportsRaw, depsRaw, hash sql.NullString
	var updatedAt sql.NullTime

	row := e.db.QueryRow(`
		SELECT summary, role, exports, dependencies, hash, updated_at
		FROM files WHERE path = ?
	`, path)

	if err := row.Scan(&summary, &role, &exportsRaw, &depsRaw, &hash, &updatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	var exports, deps []string
	if exportsRaw.Valid && exportsRaw.String != "" {
		_ = json.Unmarshal([]byte(exportsRaw.String), &exports)
	}
	if depsRaw.Valid && depsRaw.String != "" {
		_ = json.Unmarshal([]byte(depsRaw.String), &deps)
	}

	return &ai.FileAnalysis{
		Path:         path,
		Summary:      summary.String,
		Role:         role.String,
		Exports:      exports,
		Dependencies: deps,
		Hash:         hash.String,
		UpdatedAt:    updatedAt.Time,
	}, nil
}

// IngestDecisions stores PRs and decisions.
func (e *Engine) IngestDecisions(prs []github.PullRequest, decisions []ai.Decision) error {
	tx, err := e.db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmtPR, err := tx.Prepare(`
		INSERT INTO prs (number, title, url, author) VALUES (?, ?, ?, ?)
		ON CONFLICT(number) DO UPDATE SET title = excluded.title, url = excluded.url, author = excluded.author
	`)
	if err != nil {
		return err
	}
	defer stmtPR.Close()

	stmtDec, err := tx.Prepare(`
		INSERT INTO decisions (id, pr_number, title, rationale) VALUES (?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET title = excluded.title, rationale = excluded.rationale
	`)
	if err != nil {
		return err
	}
	defer stmtDec.Close()

	stmtDecFile, err := tx.Prepare(`
		INSERT OR IGNORE INTO decision_files (decision_id, file_path) VALUES (?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmtDecFile.Close()

	for i, pr := range prs {
		if _, err := stmtPR.Exec(pr.Number, pr.Title, pr.URL, pr.Author); err != nil {
			return fmt.Errorf("upsert pr %d: %w", pr.Number, err)
		}

		if i < len(decisions) {
			dec := decisions[i]
			if _, err := stmtDec.Exec(dec.ID, pr.Number, dec.Title, dec.Rationale); err != nil {
				return fmt.Errorf("upsert decision %s: %w", dec.ID, err)
			}

			for _, file := range dec.AppliesTo {
				cleanFile := strings.TrimSpace(file)
				if cleanFile != "" {
					if _, err := stmtDecFile.Exec(dec.ID, cleanFile); err != nil {
						return fmt.Errorf("link decision %s -> %s: %w", dec.ID, cleanFile, err)
					}
				}
			}
		}
	}

	return tx.Commit()
}

// --- Query Operations ---

// matchCodeownerPattern checks if a clean path matches a CODEOWNERS pattern.
func matchCodeownerPattern(filePath, pattern string) bool {
	p := strings.TrimPrefix(pattern, "/")
	p = strings.TrimSuffix(p, "*")
	p = strings.TrimSuffix(p, "/")

	if pattern == "*" || pattern == "/*" {
		return true
	}
	if p == "" {
		return false
	}
	if filePath == p {
		return true
	}
	if strings.HasPrefix(filePath, p+"/") {
		return true
	}
	return false
}

// QueryOwner finds the formal owner (from CODEOWNERS) and top 3 recent expert contributors.
func (e *Engine) QueryOwner(path string) (string, []string, error) {
	formalOwner := "Unassigned (Check CODEOWNERS)"
	var experts []string

	cleanPath := strings.TrimPrefix(path, "/")

	// 1. Check CODEOWNERS
	rows, err := e.db.Query("SELECT pattern, owners FROM codeowners")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var pattern, ownersJSON string
			if err := rows.Scan(&pattern, &ownersJSON); err == nil {
				var owners []string
				if err := json.Unmarshal([]byte(ownersJSON), &owners); err == nil {
					if matchCodeownerPattern(cleanPath, pattern) && len(owners) > 0 {
						formalOwner = strings.Join(owners, ", ")
					}
				}
			}
		}
	}

	// 2. Query top 3 experts by score
	expRows, err := e.db.Query(`
		SELECT author_name, author_email, score
		FROM expertise
		WHERE file_path = ? OR file_path LIKE ?
		ORDER BY score DESC
		LIMIT 3
	`, cleanPath, cleanPath+"/%")
	if err == nil {
		defer expRows.Close()
		for expRows.Next() {
			var name, email string
			var score float64
			if err := expRows.Scan(&name, &email, &score); err == nil {
				if name == "" {
					name = email
				}
				experts = append(experts, fmt.Sprintf("%s (%s) — Activity Score: %.2f", name, email, score))
			}
		}
	}

	return formalOwner, experts, nil
}

// QueryWhy traces the architectural purpose, decision history, and commits for a file or commit target.
func (e *Engine) QueryWhy(target string) ([]string, error) {
	var results []string
	cleanTarget := strings.TrimPrefix(target, "/")

	// 1. Check AI Architectural File Analysis (exact match or path suffix)
	var summary, role, exportsRaw, depsRaw sql.NullString
	row := e.db.QueryRow(`
		SELECT role, summary, exports, dependencies
		FROM files WHERE path = ? OR path LIKE ?
		LIMIT 1
	`, cleanTarget, "%/"+cleanTarget)

	if err := row.Scan(&role, &summary, &exportsRaw, &depsRaw); err == nil && summary.Valid && summary.String != "" {
		entry := fmt.Sprintf("🤖 AI Mind Architectural Analysis:\n  • Role:        %s\n  • Purpose:     %s", role.String, summary.String)
		var exports, deps []string
		if exportsRaw.Valid && exportsRaw.String != "" {
			_ = json.Unmarshal([]byte(exportsRaw.String), &exports)
			if len(exports) > 0 {
				entry += fmt.Sprintf("\n  • Key Symbols: %s", strings.Join(exports, ", "))
			}
		}
		if depsRaw.Valid && depsRaw.String != "" {
			_ = json.Unmarshal([]byte(depsRaw.String), &deps)
			if len(deps) > 0 {
				entry += fmt.Sprintf("\n  • Imports:     %s", strings.Join(deps, ", "))
			}
		}
		results = append(results, entry)
	}

	// 2. Check Decisions
	decRows, err := e.db.Query(`
		SELECT d.id, d.title, d.rationale, d.pr_number
		FROM decisions d
		LEFT JOIN decision_files df ON d.id = df.decision_id
		WHERE df.file_path = ? OR df.file_path LIKE ? OR d.id = ?
		LIMIT 3
	`, cleanTarget, "%/"+cleanTarget, target)
	if err == nil {
		defer decRows.Close()
		for decRows.Next() {
			var id, title, rationale string
			var prNum int
			if err := decRows.Scan(&id, &title, &rationale, &prNum); err == nil {
				results = append(results, fmt.Sprintf("Decision [%s] (PR #%d): %s\nRationale: %s", id, prNum, title, rationale))
			}
		}
	}

	// 3. Query Commits (by SHA prefix or by file touched)
	commitRows, err := e.db.Query(`
		SELECT DISTINCT c.sha, c.message, c.author_email, c.timestamp
		FROM commits c
		LEFT JOIN commit_files cf ON c.sha = cf.commit_sha
		WHERE c.sha LIKE ? OR cf.file_path = ? OR cf.file_path LIKE ?
		ORDER BY c.timestamp DESC
		LIMIT 5
	`, target+"%", cleanTarget, "%/"+cleanTarget)
	if err == nil {
		defer commitRows.Close()
		for commitRows.Next() {
			var sha, msg, author string
			var ts time.Time
			if err := commitRows.Scan(&sha, &msg, &author, &ts); err == nil {
				shortSHA := sha
				if len(shortSHA) > 7 {
					shortSHA = shortSHA[:7]
				}
				results = append(results, fmt.Sprintf("Commit [%s]: %s (%s)", shortSHA, msg, author))
			}
		}
	}

	return results, nil
}

// QueryDualOverview returns metrics for Git History and AI Mind stored in the unified SQLite database.
func (e *Engine) QueryDualOverview() (*DualOverviewStats, error) {
	stats := &DualOverviewStats{}

	_ = e.db.QueryRow("SELECT COUNT(*) FROM commits").Scan(&stats.GitCommits)
	_ = e.db.QueryRow("SELECT COUNT(*) FROM people").Scan(&stats.GitPeople)
	_ = e.db.QueryRow("SELECT COUNT(DISTINCT file_path) FROM commit_files").Scan(&stats.GitFiles)
	stats.GitLastSynced, _ = e.GetGitLastSyncedCommit()

	_ = e.db.QueryRow("SELECT COUNT(*) FROM files WHERE summary IS NOT NULL AND summary != ''").Scan(&stats.MindFiles)
	_ = e.db.QueryRow("SELECT COUNT(*) FROM decisions").Scan(&stats.MindDecisions)
	_ = e.db.QueryRow("SELECT COUNT(DISTINCT role) FROM files WHERE role IS NOT NULL AND role != ''").Scan(&stats.MindRoles)
	stats.MindLastSynced, _ = e.GetMindLastSyncedCommit()

	return stats, nil
}

// QueryOverview returns backwards-compatible map stats for MCP and tests.
func (e *Engine) QueryOverview() (map[string]int, error) {
	dual, err := e.QueryDualOverview()
	if err != nil {
		return nil, err
	}
	return map[string]int{
		"Files":         dual.GitFiles,
		"Commits":       dual.GitCommits,
		"People":        dual.GitPeople,
		"Decisions":     dual.MindDecisions,
		"AnalyzedFiles": dual.MindFiles,
	}, nil
}

// ExecuteRawSQL executes an arbitrary SQL query or table inspection against SQLite.
func (e *Engine) ExecuteRawSQL(query string) ([][]string, error) {
	trimmed := strings.TrimSpace(query)
	upper := strings.ToUpper(trimmed)

	// If query is not a SQL statement, treat it as a search across tables
	if !strings.HasPrefix(upper, "SELECT") && !strings.HasPrefix(upper, "PRAGMA") && !strings.HasPrefix(upper, "EXPLAIN") && !strings.HasPrefix(upper, "WITH") {
		return e.fallbackPrefixSearch(trimmed)
	}

	rows, err := e.db.Query(trimmed)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results [][]string
	results = append(results, cols)

	values := make([]interface{}, len(cols))
	valuePtrs := make([]interface{}, len(cols))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}
		rowStr := make([]string, len(cols))
		for i, val := range values {
			if val == nil {
				rowStr[i] = "NULL"
			} else {
				rowStr[i] = fmt.Sprintf("%v", val)
			}
		}
		results = append(results, rowStr)
	}

	return results, nil
}

// ExecuteRawCypher provides backwards compatibility for query CLI command.
func (e *Engine) ExecuteRawCypher(query string) ([][]string, error) {
	return e.ExecuteRawSQL(query)
}

func (e *Engine) fallbackPrefixSearch(term string) ([][]string, error) {
	var rows [][]string
	likeTerm := "%" + term + "%"

	// Search files
	fRows, err := e.db.Query("SELECT path, role, summary FROM files WHERE path LIKE ? OR role LIKE ? LIMIT 15", likeTerm, likeTerm)
	if err == nil {
		defer fRows.Close()
		for fRows.Next() {
			var p, r, s sql.NullString
			if err := fRows.Scan(&p, &r, &s); err == nil {
				rows = append(rows, []string{"[file] " + p.String, r.String + ": " + s.String})
			}
		}
	}

	// Search commits
	cRows, err := e.db.Query("SELECT sha, message FROM commits WHERE sha LIKE ? OR message LIKE ? LIMIT 15", likeTerm, likeTerm)
	if err == nil {
		defer cRows.Close()
		for cRows.Next() {
			var s, m string
			if err := cRows.Scan(&s, &m); err == nil {
				short := s
				if len(short) > 7 {
					short = short[:7]
				}
				rows = append(rows, []string{"[commit] " + short, m})
			}
		}
	}

	// Search decisions
	dRows, err := e.db.Query("SELECT id, title FROM decisions WHERE id LIKE ? OR title LIKE ? LIMIT 15", likeTerm, likeTerm)
	if err == nil {
		defer dRows.Close()
		for dRows.Next() {
			var id, title string
			if err := dRows.Scan(&id, &title); err == nil {
				rows = append(rows, []string{"[decision] " + id, title})
			}
		}
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i][0] < rows[j][0]
	})

	return rows, nil
}
