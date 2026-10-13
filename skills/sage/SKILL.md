---
name: sage
description: Git-native repository brain powered by embedded SQLite (.github/sage/sage.db). Enables AI agents to query code ownership, tribal knowledge, decision lineage, and repository architecture with zero context-window bloat.
---

# Sage Skill: Repository & Team Brain

When working in a codebase equipped with Sage, do not guess who wrote code, who owns it, or why an architectural pattern exists. Use the `sage` CLI or Sage MCP tools to query the local SQLite knowledge base (`.github/sage/sage.db`) directly.

## Available Agent Commands

### 1. High-Level Context (Quick repository understanding)
Run to get compact statistics and architecture knowledge without bloating your prompt context window:
```bash
sage overview
```

### 2. Ownership & Tribal Knowledge Lookup
Run to find who is formally responsible (`CODEOWNERS`) and who actually knows the code (recency-decay commit scoring):
```bash
sage who <path-to-file-or-dir>
```
*Example:* `sage who internal/auth/`  
*Result:* Returns formal owner and top recent active contributors.

### 3. Decision Lineage & "Why" Trace
Run to find out why a line of code, file, or architectural pattern was created:
```bash
sage why <file-path-or-commit-sha>
```
*Example:* `sage why internal/config/config.go`  
*Result:* Returns commit history, linked PR discussions, and synthesized decision rationale.

### 4. Custom SQL & Record Queries
Run targeted queries directly against `.github/sage/sage.db`:
```bash
sage query "SELECT name, email, recency_score FROM people ORDER BY recency_score DESC LIMIT 5"
```
Or search for keyword records:
```bash
sage query "auth"
```

## MCP Tools Integration
When Sage is running as an MCP server (`sage mcp`) in Cursor, Claude Code, or Antigravity:
- `sage_who_owns`: Pass `{"path": "internal/auth"}` to look up owners & experts.
- `sage_why`: Pass `{"target": "internal/auth/token.go"}` to trace decisions and PR discussions.
- `sage_overview`: Call to get repository graph statistics.
- `sage_query`: Pass `{"query": "..."}` for SQL queries and knowledge search.
