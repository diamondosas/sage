package graph

/*
#cgo CFLAGS: -I${SRCDIR}/../../lib-ladybug/include
#cgo LDFLAGS: -L${SRCDIR}/../../lib-ladybug/lib -llbug -lssl -lcrypto -latomic -lstdc++ -lm -lpthread
#include <stdlib.h>
#include <stdint.h>

// High-level C functions defined in ldbug.c
int ldbug_open(const char* path, void** out_db, void** out_conn);
void ldbug_close(void* db, void* conn);
int ldbug_get(void* conn, const char* key, char** out_val);
int ldbug_set(void* conn, const char* key, const char* val);
int ldbug_delete(void* conn, const char* key);
int ldbug_begin(void* conn);
int ldbug_commit(void* conn);
int ldbug_rollback(void* conn);
int ldbug_count_prefix(void* conn, const char* prefix, int* out_count);
int ldbug_scan_prefix(void* conn, const char* prefix, int limit, char*** out_keys, char*** out_vals, int* out_count);
int ldbug_query_cypher(void* conn, const char* cypher, char*** out_cells, int* out_rows, int* out_cols, char** out_err);
void ldbug_free(void* ptr);
void ldbug_free_string_array(char** arr, int count);
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unsafe"

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

// dbHandle wraps low-level LadybugDB database and connection pointers.
type dbHandle struct {
	db   unsafe.Pointer
	conn unsafe.Pointer
	mu   sync.RWMutex
}

func openDB(path string) (*dbHandle, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	var dbPtr, connPtr unsafe.Pointer
	if C.ldbug_open(cPath, &dbPtr, &connPtr) != 0 {
		return nil, fmt.Errorf("failed to open LadybugDB at %s", path)
	}
	return &dbHandle{db: dbPtr, conn: connPtr}, nil
}

func (h *dbHandle) close() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.conn != nil || h.db != nil {
		C.ldbug_close(h.db, h.conn)
		h.db = nil
		h.conn = nil
	}
	return nil
}

func (h *dbHandle) get(key string) (string, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.conn == nil {
		return "", fmt.Errorf("database not open")
	}
	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))

	var cVal *C.char
	status := C.ldbug_get(h.conn, cKey, &cVal)
	if status == 1 || cVal == nil {
		return "", nil // key not found
	}
	if status < 0 {
		return "", fmt.Errorf("error reading key %s", key)
	}
	val := C.GoString(cVal)
	C.ldbug_free(unsafe.Pointer(cVal))
	return val, nil
}

func (h *dbHandle) set(key, val string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.conn == nil {
		return fmt.Errorf("database not open")
	}
	cKey := C.CString(key)
	cVal := C.CString(val)
	defer C.free(unsafe.Pointer(cKey))
	defer C.free(unsafe.Pointer(cVal))

	if C.ldbug_set(h.conn, cKey, cVal) != 0 {
		return fmt.Errorf("error writing key %s", key)
	}
	return nil
}

func (h *dbHandle) delete(key string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.conn == nil {
		return fmt.Errorf("database not open")
	}
	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))

	if C.ldbug_delete(h.conn, cKey) != 0 {
		return fmt.Errorf("error deleting key %s", key)
	}
	return nil
}

func (h *dbHandle) countPrefix(prefix string) (int, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.conn == nil {
		return 0, fmt.Errorf("database not open")
	}
	cPrefix := C.CString(prefix)
	defer C.free(unsafe.Pointer(cPrefix))

	var count C.int
	if C.ldbug_count_prefix(h.conn, cPrefix, &count) != 0 {
		return 0, fmt.Errorf("error counting prefix %s", prefix)
	}
	return int(count), nil
}

func (h *dbHandle) scanPrefix(prefix string, limit int) (map[string]string, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.conn == nil {
		return nil, fmt.Errorf("database not open")
	}
	cPrefix := C.CString(prefix)
	defer C.free(unsafe.Pointer(cPrefix))

	var cKeys, cVals **C.char
	var cCount C.int
	if C.ldbug_scan_prefix(h.conn, cPrefix, C.int(limit), &cKeys, &cVals, &cCount) != 0 {
		return nil, fmt.Errorf("error scanning prefix %s", prefix)
	}

	count := int(cCount)
	res := make(map[string]string, count)
	if count > 0 && cKeys != nil && cVals != nil {
		keysSlice := (*[1 << 28]*C.char)(unsafe.Pointer(cKeys))[:count:count]
		valsSlice := (*[1 << 28]*C.char)(unsafe.Pointer(cVals))[:count:count]
		for i := 0; i < count; i++ {
			k := C.GoString(keysSlice[i])
			v := C.GoString(valsSlice[i])
			res[k] = v
		}
		C.ldbug_free_string_array(cKeys, cCount)
		C.ldbug_free_string_array(cVals, cCount)
	}
	return res, nil
}

func (h *dbHandle) queryCypher(cypher string) ([][]string, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.conn == nil {
		return nil, fmt.Errorf("database not open")
	}
	cQuery := C.CString(cypher)
	defer C.free(unsafe.Pointer(cQuery))

	var cCells **C.char
	var cRows, cCols C.int
	var cErr *C.char

	if C.ldbug_query_cypher(h.conn, cQuery, &cCells, &cRows, &cCols, &cErr) != 0 {
		errStr := "query failed"
		if cErr != nil {
			errStr = C.GoString(cErr)
			C.ldbug_free(unsafe.Pointer(cErr))
		}
		return nil, fmt.Errorf("%s", errStr)
	}

	rows := int(cRows)
	cols := int(cCols)
	out := make([][]string, rows)
	if rows > 0 && cols > 0 && cCells != nil {
		cellsSlice := (*[1 << 28]*C.char)(unsafe.Pointer(cCells))[: rows*cols : rows*cols]
		for r := 0; r < rows; r++ {
			out[r] = make([]string, cols)
			for c := 0; c < cols; c++ {
				out[r][c] = C.GoString(cellsSlice[r*cols+c])
			}
		}
		C.ldbug_free_string_array(cCells, C.int(rows*cols))
	}
	return out, nil
}

// Engine manages dual embedded LadybugDB instances:
// 1. gitDB (git.lddb) - Git history, file touch index, and recency-decay expertise
// 2. mindDB (sage.lddb) - AI-synthesized codebase knowledge and architectural roles
type Engine struct {
	gitDB   *dbHandle
	mindDB  *dbHandle
	baseDir string
	mu      sync.RWMutex
}

// NewEngine initializes the dual LadybugDB instances at dbPath.
// Defaults to filepath.Join(".github", "sage").
func NewEngine(dbPath string) (*Engine, error) {
	if dbPath == "" {
		dbPath = filepath.Join(".github", "sage")
	}
	if err := os.MkdirAll(dbPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create db directory %s: %w", dbPath, err)
	}

	// Filenames for the dual embedded LadybugDB database files (.lddb)
	// CHANGE HERE: Modify filenames for git and AI mind graphs if desired
	gitPath := filepath.Join(dbPath, "git.lddb")
	mindPath := filepath.Join(dbPath, "sage.lddb")

	gDB, err := openDB(gitPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open Git LadybugDB at %s: %w", gitPath, err)
	}

	mDB, err := openDB(mindPath)
	if err != nil {
		_ = gDB.close()
		return nil, fmt.Errorf("failed to open AI Mind LadybugDB at %s: %w", mindPath, err)
	}

	return &Engine{
		gitDB:   gDB,
		mindDB:  mDB,
		baseDir: dbPath,
	}, nil
}

// Close closes both LadybugDB database instances.
func (e *Engine) Close() error {
	if e == nil {
		return nil
	}
	var errs []string
	if e.gitDB != nil {
		if err := e.gitDB.close(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if e.mindDB != nil {
		if err := e.mindDB.close(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("error closing database(s): %s", strings.Join(errs, "; "))
	}
	return nil
}

// InitSchema ensures tables are initialized (handled automatically in ldbug_open).
func (e *Engine) InitSchema() error {
	return nil
}

// --- Sync Metadata Tracking ---

// GetGitLastSyncedCommit returns the SHA of the last commit synced to git.lddb.
func (e *Engine) GetGitLastSyncedCommit() (string, error) {
	if e.gitDB == nil {
		return "", fmt.Errorf("git database not initialized")
	}
	return e.gitDB.get("meta:last_synced_commit")
}

// SetGitLastSyncedCommit records the latest commit SHA in git.lddb.
func (e *Engine) SetGitLastSyncedCommit(sha string) error {
	if e.gitDB == nil {
		return fmt.Errorf("git database not initialized")
	}
	return e.gitDB.set("meta:last_synced_commit", sha)
}

// GetMindLastSyncedCommit returns the SHA of the last commit synced to sage.lddb.
func (e *Engine) GetMindLastSyncedCommit() (string, error) {
	if e.mindDB == nil {
		return "", fmt.Errorf("mind database not initialized")
	}
	return e.mindDB.get("meta:last_synced_commit")
}

// SetMindLastSyncedCommit records the latest commit SHA in sage.lddb.
func (e *Engine) SetMindLastSyncedCommit(sha string) error {
	if e.mindDB == nil {
		return fmt.Errorf("mind database not initialized")
	}
	return e.mindDB.set("meta:last_synced_commit", sha)
}

// --- Git Graph Operations (git.lddb) ---

// IngestGitHistory stores persons, commits, touched file indexes, and accumulates expertise.
func (e *Engine) IngestGitHistory(commits []git.Commit, expertise map[string][]git.ContributorExpertise) error {
	if e.gitDB == nil {
		return fmt.Errorf("git database not initialized")
	}

	C.ldbug_begin(e.gitDB.conn)
	for _, c := range commits {
		// 1. Person
		personKey := fmt.Sprintf("person:%s", strings.ToLower(c.AuthorEmail))
		p := StoredPerson{Email: c.AuthorEmail, Name: c.AuthorName}
		if pBytes, err := json.Marshal(p); err == nil {
			_ = e.gitDB.set(personKey, string(pBytes))
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
			_ = e.gitDB.set(commitKey, string(cBytes))
		}

		// 3. Index: file -> commit
		for _, f := range c.Files {
			if strings.TrimSpace(f) == "" {
				continue
			}
			idxKey := fmt.Sprintf("file_commit:%s:%s", f, c.SHA)
			_ = e.gitDB.set(idxKey, c.SHA)

			// Record file existence
			fileKey := fmt.Sprintf("file:%s", f)
			_ = e.gitDB.set(fileKey, f)
		}
	}

	// 4. Ingest & merge calculated contributor expertise
	for file, contribs := range expertise {
		for _, contrib := range contribs {
			exp := StoredExpertise{
				AuthorEmail: contrib.AuthorEmail,
				AuthorName:  contrib.AuthorName,
				Path:        file,
				Score:       contrib.Score,
				CommitCount: contrib.CommitCount,
			}
			expKey := fmt.Sprintf("expertise:%s:%s", file, strings.ToLower(contrib.AuthorEmail))

			// Check if existing record exists to accumulate score incrementally
			if existingStr, err := e.gitDB.get(expKey); err == nil && existingStr != "" {
				var existing StoredExpertise
				if err := json.Unmarshal([]byte(existingStr), &existing); err == nil {
					exp.Score += existing.Score
					exp.CommitCount += existing.CommitCount
				}
			}

			if expBytes, err := json.Marshal(exp); err == nil {
				_ = e.gitDB.set(expKey, string(expBytes))
			}
		}
	}
	C.ldbug_commit(e.gitDB.conn)
	return nil
}

// IngestCodeowners stores CODEOWNERS rules in git.lddb.
func (e *Engine) IngestCodeowners(rules []git.OwnershipRule) error {
	if e.gitDB == nil {
		return fmt.Errorf("git database not initialized")
	}
	C.ldbug_begin(e.gitDB.conn)
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
		_ = e.gitDB.set(key, string(data))
	}
	C.ldbug_commit(e.gitDB.conn)
	return nil
}

// --- AI Mind Graph Operations (sage.lddb) ---

// GetStoredFileHash returns the stored content hash for a file from sage.lddb.
func (e *Engine) GetStoredFileHash(path string) (string, error) {
	if e.mindDB == nil {
		return "", fmt.Errorf("ai mind database not initialized")
	}
	return e.mindDB.get(fmt.Sprintf("file_hash:%s", path))
}

// IngestFileAnalysis stores AI file analysis and updates the file hash index in sage.lddb.
func (e *Engine) IngestFileAnalysis(analyses []ai.FileAnalysis) error {
	if e.mindDB == nil {
		return fmt.Errorf("ai mind database not initialized")
	}
	C.ldbug_begin(e.mindDB.conn)
	for _, a := range analyses {
		data, err := json.Marshal(a)
		if err != nil {
			continue
		}
		// 1. File analysis record
		key := fmt.Sprintf("file_summary:%s", a.Path)
		_ = e.mindDB.set(key, string(data))

		// 2. File hash for change detection
		hashKey := fmt.Sprintf("file_hash:%s", a.Path)
		_ = e.mindDB.set(hashKey, a.Hash)
		// 3. Role index
		if a.Role != "" {
			roleKey := fmt.Sprintf("role:%s:%s", a.Role, a.Path)
			_ = e.mindDB.set(roleKey, a.Path)
		}
	}
	C.ldbug_commit(e.mindDB.conn)
	return nil
}

// DeleteFileAnalysis removes file analyses for deleted files from sage.lddb.
func (e *Engine) DeleteFileAnalysis(paths []string) error {
	if e.mindDB == nil {
		return fmt.Errorf("ai mind database not initialized")
	}
	C.ldbug_begin(e.mindDB.conn)
	for _, p := range paths {
		_ = e.mindDB.delete(fmt.Sprintf("file_summary:%s", p))
		_ = e.mindDB.delete(fmt.Sprintf("file_hash:%s", p))
	}
	C.ldbug_commit(e.mindDB.conn)
	return nil
}

// GetFileAnalysis returns the stored AI analysis for a file path.
func (e *Engine) GetFileAnalysis(path string) (*ai.FileAnalysis, error) {
	if e.mindDB == nil {
		return nil, fmt.Errorf("ai mind database not initialized")
	}
	val, err := e.mindDB.get(fmt.Sprintf("file_summary:%s", path))
	if err != nil || val == "" {
		return nil, err
	}
	var a ai.FileAnalysis
	if err := json.Unmarshal([]byte(val), &a); err != nil {
		return nil, err
	}
	return &a, nil
}

// IngestDecisions stores PRs and decisions in sage.lddb.
func (e *Engine) IngestDecisions(prs []github.PullRequest, decisions []ai.Decision) error {
	if e.mindDB == nil {
		return fmt.Errorf("ai mind database not initialized")
	}
	C.ldbug_begin(e.mindDB.conn)
	for i, pr := range prs {
		prKey := fmt.Sprintf("pr:%d", pr.Number)
		storedPR := StoredPR{
			Number: pr.Number,
			Title:  pr.Title,
			URL:    pr.URL,
			Author: pr.Author,
		}
		if prBytes, err := json.Marshal(storedPR); err == nil {
			_ = e.mindDB.set(prKey, string(prBytes))
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
				_ = e.mindDB.set(decKey, string(decBytes))

				prDecKey := fmt.Sprintf("pr_decision:%d:%s", pr.Number, dec.ID)
				_ = e.mindDB.set(prDecKey, dec.ID)
			}
		}
	}
	C.ldbug_commit(e.mindDB.conn)
	return nil
}

// --- Query Operations ---

// QueryOwner finds formal owner and top 3 recent experts for a path from git.lddb.
func (e *Engine) QueryOwner(path string) (string, []string, error) {
	if e.gitDB == nil {
		return "", nil, fmt.Errorf("git database not initialized")
	}
	formalOwner := "Unassigned (Check CODEOWNERS)"
	var experts []string

	// 1. Look for matching ownership rule
	// CHANGE HERE: Max ownership rules scanned (default: 1000)
	rulesMap, err := e.gitDB.scanPrefix("ownership:", 1000)
	if err != nil {
		return "", nil, err
	}
	cleanPath := strings.TrimPrefix(path, "/")
	for _, val := range rulesMap {
		var rule StoredOwnership
		if err := json.Unmarshal([]byte(val), &rule); err == nil {
			cleanPattern := strings.TrimPrefix(rule.Pattern, "/")
			cleanPattern = strings.TrimSuffix(cleanPattern, "*")
			cleanPattern = strings.TrimSuffix(cleanPattern, "/")
			if cleanPattern == "" || strings.HasPrefix(cleanPath, cleanPattern) || cleanPath == cleanPattern {
				if len(rule.Owners) > 0 {
					formalOwner = strings.Join(rule.Owners, ", ")
				}
			}
		}
	}

	// 2. Aggregate expertise scores for the path
	// CHANGE HERE: Max expertise records scanned (default: 10000)
	expMap, err := e.gitDB.scanPrefix("expertise:", 10000)
	if err != nil {
		return formalOwner, experts, err
	}

	scoreMap := make(map[string]*StoredExpertise)
	for _, val := range expMap {
		var exp StoredExpertise
		if err := json.Unmarshal([]byte(val), &exp); err == nil {
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
	}

	var list []*StoredExpertise
	for _, exp := range scoreMap {
		list = append(list, exp)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Score > list[j].Score
	})

	// CHANGE HERE: Number of top recent experts returned (default: 3)
	for i := 0; i < len(list) && i < 3; i++ {
		exp := list[i]
		name := exp.AuthorName
		if name == "" {
			name = exp.AuthorEmail
		}
		experts = append(experts, fmt.Sprintf("%s (%s) — Activity Score: %.2f", name, exp.AuthorEmail, exp.Score))
	}

	return formalOwner, experts, nil
}

// QueryWhy traces the origin, AI architectural purpose, and commit lineage.
func (e *Engine) QueryWhy(target string) ([]string, error) {
	var results []string

	// 1. Check AI Mind (sage.lddb) for architectural file summary and role
	if e.mindDB != nil {
		if val, err := e.mindDB.get(fmt.Sprintf("file_summary:%s", target)); err == nil && val != "" {
			var a ai.FileAnalysis
			if err := json.Unmarshal([]byte(val), &a); err == nil {
				entry := fmt.Sprintf("🤖 AI Mind Architectural Analysis:\n  • Role:        %s\n  • Purpose:     %s", a.Role, a.Summary)
				if len(a.Exports) > 0 {
					entry += fmt.Sprintf("\n  • Key Symbols: %s", strings.Join(a.Exports, ", "))
				}
				if len(a.Dependencies) > 0 {
					entry += fmt.Sprintf("\n  • Imports:     %s", strings.Join(a.Dependencies, ", "))
				}
				results = append(results, entry)
			}
		}
	}

	// 2. Query Git history and decisions from gitDB
	if e.gitDB != nil {
		// CHANGE HERE: Max commits scanned when tracing file/commit origins (default: 5000)
		commitsMap, err := e.gitDB.scanPrefix("commit:", 5000)
		if err == nil {
			var matchedCommits []StoredCommit
			for _, val := range commitsMap {
				var sc StoredCommit
				if err := json.Unmarshal([]byte(val), &sc); err == nil {
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
			}

			sort.Slice(matchedCommits, func(i, j int) bool {
				return matchedCommits[i].Timestamp.After(matchedCommits[j].Timestamp)
			})

			// CHANGE HERE: Number of matching commits shown in QueryWhy (default: 5)
			for i, sc := range matchedCommits {
				if i >= 5 {
					break
				}
				sha := sc.SHA
				if len(sha) > 7 {
					sha = sha[:7]
				}
				results = append(results, fmt.Sprintf("Commit [%s]: %s (%s)", sha, sc.Message, sc.Author))
			}
		}
	}

	return results, nil
}

// QueryDualOverview returns metrics for both git.lddb and sage.lddb.
func (e *Engine) QueryDualOverview() (*DualOverviewStats, error) {
	stats := &DualOverviewStats{}

	if e.gitDB != nil {
		stats.GitFiles, _ = e.gitDB.countPrefix("file:")
		stats.GitCommits, _ = e.gitDB.countPrefix("commit:")
		stats.GitPeople, _ = e.gitDB.countPrefix("person:")
		stats.GitLastSynced, _ = e.GetGitLastSyncedCommit()
	}

	if e.mindDB != nil {
		stats.MindFiles, _ = e.mindDB.countPrefix("file_summary:")
		stats.MindDecisions, _ = e.mindDB.countPrefix("decision:")
		stats.MindRoles, _ = e.mindDB.countPrefix("role:")
		stats.MindLastSynced, _ = e.GetMindLastSyncedCommit()
	}

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

// ExecuteRawCypher executes raw Cypher queries or key prefix scans across LadybugDB databases.
func (e *Engine) ExecuteRawCypher(query string) ([][]string, error) {
	trimmed := strings.TrimSpace(query)
	upper := strings.ToUpper(trimmed)

	// If the query is an explicit Cypher query, execute it directly
	if strings.HasPrefix(upper, "MATCH") || strings.HasPrefix(upper, "CREATE") || strings.HasPrefix(upper, "RETURN") {
		if e.gitDB != nil {
			rows, err := e.gitDB.queryCypher(trimmed)
			if err == nil {
				return rows, nil
			}
		}
		if e.mindDB != nil {
			return e.mindDB.queryCypher(trimmed)
		}
		return nil, fmt.Errorf("no database available")
	}

	// Otherwise, fallback to prefix scan for key-value inspection
	var rows [][]string

	// Scan gitDB
	// CHANGE HERE: Maximum results returned per database in prefix query mode
	maxRows := 25
	if e.gitDB != nil {
		gitEntries, _ := e.gitDB.scanPrefix(trimmed, maxRows)
		for k, v := range gitEntries {
			rows = append(rows, []string{"[git] " + k, v})
			if len(rows) >= maxRows {
				break
			}
		}
	}

	// Scan mindDB
	if e.mindDB != nil && len(rows) < 50 {
		mindEntries, _ := e.mindDB.scanPrefix(trimmed, 50-len(rows))
		for k, v := range mindEntries {
			rows = append(rows, []string{"[sage] " + k, v})
			if len(rows) >= 50 {
				break
			}
		}
	}

	return rows, nil
}
