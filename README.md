# Sage 🧠

**Sage** is a Git-native Company Brain powered by an embedded **BadgerDB** engine. It connects your codebase files with Git commit history, human decisions, and CODEOWNERS rules directly on disk—no Docker, no server daemon required (just like SQLite). Developers and AI coding assistants can query who owns code, why architectural patterns exist, and how code evolved over time without dumping huge files into prompt windows.

---

## Prerequisites

- **Go 1.22+** (Only needed if building from source)
- **Git**
*(No Docker or external database required!)*

---

## Installation

### Recommended: NPM & AI Agent Skill Setup (Like Caveman)
Installs both the Sage CLI globally and detects your AI editors (Cursor, Claude Code, Antigravity, Copilot, Windsurf, Codex) to install the `sage` agent skill:

```bash
npm install -g sage-brain && sage setup --install
```

### Option 2: Build from Source (Go)
---

## Quick Start (2 Steps)

### Step 1: Initialize Sage in Your Repository
Navigate to any git project and run:
```bash
sage init
```
This prompts for a GitHub token (saving it to `.env` / `.gitignore`), generates `.github/workflows/sage-sync.yml` to automatically keep the BadgerDB graph synced on PRs and pushes, and writes `AGENTS.md` for AI coding assistants.

### Step 2: Index Your Git History
```bash
sage sync
```
Sage scans your git commits, calculates contributor activity scores, pulls merged pull requests (if connected to GitHub), and saves the graph directly into embedded BadgerDB in `.github/sage`.

---

## CLI Commands

### 1. Find Code Ownership & Experts
Find the formal owner (from `CODEOWNERS`) and top recent active contributors (weighted by recency):
```bash
sage who internal/graph/falkordb.go
```

### 2. Trace Why Code Exists
Trace a file or commit back to originating PR debates and architectural decisions:
```bash
sage why internal/config/config.go
```

### 3. Repository Overview
Get a high-level summary of your knowledge graph without context bloat:
```bash
sage overview
```

### 4. Custom Cypher Queries
Run direct graph queries against FalkorDB:
```bash
sage query "MATCH (p:Person)-[:AUTHORED]->(c:Commit) RETURN p.name, count(c) ORDER BY count(c) DESC LIMIT 5"
```

### 5. Run MCP Server (for AI Assistants)
Start the Model Context Protocol stdio server for Cursor, Claude Code, or Antigravity:
```bash
sage mcp
```

---

## AI Assistants Setup (Cursor, Claude Code, Antigravity)

Add Sage to your MCP configuration file (e.g. `~/.cursor/mcp.json` or `claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "sage": {
      "command": "sage",
      "args": ["mcp"],
      "env": {
        "FALKORDB_HOST": "127.0.0.1",
        "FALKORDB_PORT": "6379"
      }
    }
  }
}
```

Now your AI assistant has access to `sage_who_owns`, `sage_why`, `sage_overview`, and `sage_query` directly in chat!

---

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `FALKORDB_HOST` | `127.0.0.1` | FalkorDB server host |
| `FALKORDB_PORT` | `6379` | FalkorDB server port |
| `FALKORDB_GRAPH` | `sage` | Graph key name in FalkorDB |
| `GITHUB_TOKEN` | *(empty)* | Optional GitHub token to fetch PR discussions |
| `LLM_PROVIDER` | `google` | Provider for PR decision synthesis (`google`, `openai`, `anthropic`, `ollama`) |
| `LLM_MODEL` | `gemini-2.5-flash`| Model name (e.g., `gpt-4o-mini`, `llama3`) |
| `GEMINI_API_KEY` | *(empty)* | Google Gemini API key |
| `OPENAI_API_KEY` | *(empty)* | OpenAI API key |
| `ANTHROPIC_API_KEY` | *(empty)* | Anthropic API key |
| `SAGE_MAX_COMMITS` | `500` | Number of recent commits to scan |
| `SAGE_DECAY_DAYS` | `60` | Half-life in days for contributor recency scoring |

---

## License
MIT
