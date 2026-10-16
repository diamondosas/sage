---
name: sage
description: Git-native repository brain powered by embedded SQLite (.github/sage/sage.db). Enables AI agents to inspect architecture domain clusters, file ownership, decision lineage, and repository architecture with zero context-window bloat.
---

# Sage Skill: Repository & Team Brain

When working in a codebase equipped with Sage, do not guess how modules connect, who owns code, or why an architectural pattern exists. Use the `sage` CLI or local SQLite knowledge base (`.github/sage/sage.db`) directly.

## Available Agent Commands

### 1. Interactive Architecture Visualizer (`sage ui`)
Launch the local web visualizer and embedded REST API:
```bash
sage ui
```
Flags:
- `--port <number>`: Port to run the server on (default: `8080` or next open port).
- `--no-browser`: Start the server without automatically launching the browser (ideal for headless environments, CI/CD, and background agent audits).

**API Endpoints for Agents:**
- `GET http://127.0.0.1:8080/api/graph`: Returns high-level architectural domain clusters, member files, and inter-cluster dependency edges.
- `GET http://127.0.0.1:8080/api/file?path=<path>`: Returns detailed inspection data for any file (architectural role, purpose summary, exported symbols, imports, formal CODEOWNER, recent expert contributors, and Why lineage).

### 2. High-Level Context (Quick repository understanding)
Run to get compact statistics and architecture knowledge without bloating your prompt context window:
```bash
sage overview
```

### 3. Custom SQL & Record Queries
Run targeted queries directly against `.github/sage/sage.db`:
```bash
sage query "SELECT name, email, recency_score FROM people ORDER BY recency_score DESC LIMIT 5"
```
Or search for keyword records:
```bash
sage query "auth"
```
