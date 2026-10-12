---
name: sage
description: Git-native repository brain powered by FalkorDB. Enables AI agents to query code ownership, tribal knowledge, decision lineage, and repository architecture with zero context-window bloat.
---

# Sage Skill: Repository & Company Brain

When working in a codebase equipped with Sage, do not guess who wrote code, who owns it, or why an architectural pattern exists. Use the `sage` CLI or Sage MCP tools to query the underlying FalkorDB graph directly.

## Available Agent Commands

### 1. High-Level Context (Get understanding of everything at once)
Run to get compact statistics and top domains without filling your prompt:
```bash
sage overview
```

### 2. Ownership & Tribal Knowledge Lookup
Run to find who is formally responsible (CODEOWNERS) and who actually knows the code (recent commit recency score):
```bash
sage who <path-to-file-or-dir>
```
*Example:* `sage who internal/auth/`
*Result:* Returns formal owner team and top 3 active contributors.

### 3. Decision Lineage & "Why" Trace
Run to find out why a line of code, file, or architectural pattern was created:
```bash
sage why <file-path-or-commit-sha>
```
*Example:* `sage why internal/config/config.go`
*Result:* Returns commit history, linked PR discussions, and synthesized decision rationale.

### 4. Custom Graph Traversals (Cypher)
Run targeted Cypher queries directly against FalkorDB:
```bash
sage query "MATCH (p:Person)-[:AUTHORED]->(c:Commit)-[:TOUCHED]->(f:File {path: 'main.go'}) RETURN p.name, c.message"
```

## MCP Tools Integration
If Sage is configured as an MCP server in your coding environment (Cursor, Claude Code, Antigravity):
- `sage_who_owns`: Pass `{"path": "internal/auth"}` to look up owners.
- `sage_why`: Pass `{"target": "internal/auth/token.go"}` to trace decisions.
- `sage_overview`: Call to get repository graph statistics.
- `sage_query`: Pass `{"cypher": "..."}` for multi-hop graph queries.
