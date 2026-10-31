---
name: sage
description: Visual architecture mapping of codebase structure, modules, dependencies, and technical debt into SAGE.md. Use when the user asks to map architecture, visualize the codebase, inspect structural tech debt, update SAGE.md, or run the Sage CLI diagram viewer.
---

# Sage Skill: Architecture Mapping

Map codebase architecture into `SAGE.md` so human users can visually inspect structure and technical debt in the `sage` browser UI.


## Mission & Rules
1. **Never guess**: Don't assume facts about the codebase. Do not invent file paths, dependencies, or patterns if unsure. Check git status or directory trees first.
2. **Concise explanations**: Keep commentary minimal and focused on structure.
3. **Reference documentation**: See [references/language-syntax.md](references/language-syntax.md) for full syntax, shapes, and color rules.

## SAGE.md Syntax Quick Reference
Architecture maps in `SAGE.md` are specified using standard D2 code blocks:

```d2
direction: right

cli: "CLI Layer (cmd/sage)" {
  user: "Terminal User" {
    shape: circle
    style.fill: "#14532d"
    style.stroke: "#4ade80"
  }
  app: "Cobra CLI" {
    shape: rectangle
    style.fill: "#1e3a8a"
    style.stroke: "#60a5fa"
  }
  gate: "Decision Gate" {
    shape: diamond
    style.fill: "#713f12"
    style.stroke: "#facc15"
  }
}

storage: "Storage Layer (internal/storage)" {
  db: "Database Store" {
    shape: cylinder
    style.fill: "#581c87"
    style.stroke: "#c084fc"
  }
  worker: "Task Worker" {
    shape: queue
    style.fill: "#7c2d12"
    style.stroke: "#fb923c"
  }
}

cli.user -> cli.app
cli.app -> cli.gate
cli.gate -> storage.db: "Dispatches query"
cli.gate <-> storage.worker: "Bidirectional stream"
```

## Incremental Update Protocol
When code changes after `SAGE.md` exists:
1. **Never regenerate the whole file.**
2. Run `git status` and `git diff` to locate modified files and packages.
3. Locate corresponding `// group:` or node IDs in `SAGE.md`.
4. Surgically update or append only the touched nodes and connections.
5. Remind the user to run `sage ui` in their terminal to view the interactive diagram. If `sage` is not installed on their system, tell them to install it via:
   `curl -fsSL https://raw.githubusercontent.com/diamondosas/sage/main/assets/install.sh | sh` (or `npx sage-codebase`).
