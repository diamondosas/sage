# 🤖 Sage Agent Skills Setup Guide

> **Give your AI pair programmer (Cursor, Claude Code, Antigravity, GitHub Copilot, Windsurf) the ability to autonomously look up code owners and PR decisions with zero context-window bloat.**

---

## ⚡ Quick 1-Command Setup

Sage includes an automated installer that detects all installed AI assistants on your system and places the `SKILL.md` file in their respective global skill directories:

```bash
# Using Sage CLI
sage setup -i

# Or using npx
npx sage-brain setup
```

The script automatically detects and configures:
- **Google Antigravity & Gemini CLI (`agy`)**
- **Cursor**
- **Claude Code (`claude`)**
- **GitHub Copilot**
- **Windsurf / Codeium**
- **Codex CLI**

---

## 📂 Manual Setup by AI Assistant

If you want to configure your assistant manually, place `skills/sage/SKILL.md` into your assistant's directory:

| Assistant | Destination Path | Terminal Install Command |
|---|---|---|
| **Antigravity / Gemini CLI (`agy`)** | `~/.gemini/config/skills/sage/SKILL.md` | `mkdir -p ~/.gemini/config/skills/sage && cp skills/sage/SKILL.md ~/.gemini/config/skills/sage/` |
| **Cursor** | `~/.cursor/skills/sage/SKILL.md` (or `.cursorrules`) | `mkdir -p ~/.cursor/skills/sage && cp skills/sage/SKILL.md ~/.cursor/skills/sage/` |
| **Claude Code** | `~/.claude/skills/sage/SKILL.md` | `mkdir -p ~/.claude/skills/sage && cp skills/sage/SKILL.md ~/.claude/skills/sage/` |
| **GitHub Copilot** | `~/.copilot/skills/sage/SKILL.md` (or `.github/copilot-instructions.md`) | `mkdir -p ~/.copilot/skills/sage && cp skills/sage/SKILL.md ~/.copilot/skills/sage/` |
| **Windsurf** | `~/.codeium/windsurf/skills/sage/SKILL.md` | `mkdir -p ~/.codeium/windsurf/skills/sage && cp skills/sage/SKILL.md ~/.codeium/windsurf/skills/sage/` |
| **Codex CLI** | `~/.codex/skills/sage/SKILL.md` | `mkdir -p ~/.codex/skills/sage && cp skills/sage/SKILL.md ~/.codex/skills/sage/` |

---

## 📋 Full `SKILL.md` Content (Copy & Paste)

```markdown
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

### 2. Interactive Architecture Visualizer & REST API
Launch the local web visualizer and embedded REST endpoints:
```bash
sage ui
```
Flags:
- `--port <number>`: Port to run the server on (default: `8080` or next open port).
- `--no-browser`: Start the server without automatically launching the browser.

**API Endpoints:**
- `GET http://127.0.0.1:8080/api/graph`: Returns domain clusters, member files, and dependency edges.
- `GET http://127.0.0.1:8080/api/file?path=<path>`: Returns detailed inspection data (role, exports, dependencies, owner, experts, decision lineage).

### 3. Custom SQL & Record Queries
Run targeted queries directly against `.github/sage/sage.db`:
```bash
sage query "SELECT name, email, recency_score FROM people ORDER BY recency_score DESC LIMIT 5"
```
Or search for keyword records:
```bash
sage query "auth"
```

