package graph

// SchemaDDL defines the unified single-file SQLite schema for Sage.
const SchemaDDL = `
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA foreign_keys = ON;
PRAGMA busy_timeout = 5000;

CREATE TABLE IF NOT EXISTS meta (
    key TEXT PRIMARY KEY,
    val TEXT
);

CREATE TABLE IF NOT EXISTS people (
    email TEXT PRIMARY KEY,
    name TEXT
);

CREATE TABLE IF NOT EXISTS commits (
    sha TEXT PRIMARY KEY,
    author_email TEXT,
    message TEXT,
    timestamp DATETIME
);
CREATE INDEX IF NOT EXISTS idx_commits_author ON commits(author_email);
CREATE INDEX IF NOT EXISTS idx_commits_timestamp ON commits(timestamp DESC);

CREATE TABLE IF NOT EXISTS files (
    path TEXT PRIMARY KEY,
    summary TEXT,
    role TEXT,
    exports TEXT,
    dependencies TEXT,
    hash TEXT,
    updated_at DATETIME
);
CREATE INDEX IF NOT EXISTS idx_files_role ON files(role);

CREATE TABLE IF NOT EXISTS commit_files (
    commit_sha TEXT,
    file_path TEXT,
    PRIMARY KEY(commit_sha, file_path)
);
CREATE INDEX IF NOT EXISTS idx_commit_files_path ON commit_files(file_path);

CREATE TABLE IF NOT EXISTS expertise (
    file_path TEXT,
    author_email TEXT,
    author_name TEXT,
    score REAL,
    commit_count INT,
    PRIMARY KEY(file_path, author_email)
);
CREATE INDEX IF NOT EXISTS idx_expertise_path_score ON expertise(file_path, score DESC);

CREATE TABLE IF NOT EXISTS codeowners (
    pattern TEXT PRIMARY KEY,
    owners TEXT
);

CREATE TABLE IF NOT EXISTS prs (
    number INT PRIMARY KEY,
    title TEXT,
    url TEXT,
    author TEXT
);

CREATE TABLE IF NOT EXISTS decisions (
    id TEXT PRIMARY KEY,
    pr_number INT,
    title TEXT,
    rationale TEXT
);
CREATE INDEX IF NOT EXISTS idx_decisions_pr ON decisions(pr_number);

CREATE TABLE IF NOT EXISTS decision_files (
    decision_id TEXT,
    file_path TEXT,
    PRIMARY KEY(decision_id, file_path)
);
CREATE INDEX IF NOT EXISTS idx_decision_files_path ON decision_files(file_path);
`
