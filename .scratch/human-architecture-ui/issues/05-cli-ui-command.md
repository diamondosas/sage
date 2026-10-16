# Ticket 05: CLI `sage ui` Command & Browser Launcher

Status: resolved
Blocked by: 04

## What it delivers

Adds the CLI command and connects everything end-to-end:
- Implements `cmd/sage/cli/ui.go` defining `sage ui`.
- Supports `--port` (default: 8080 or next open port) and `--no-browser`.
- Cross-platform browser launcher (Linux, macOS, Windows).
- Updates `skills/sage/SKILL.md` and repo documentation to reflect `sage ui` and remove references to removed commands (`who`, `why`, `mcp`).
- Verified with end-to-end command tests.

## Comments

- Implemented `cmd/sage/cli/ui.go` supporting `sage ui` with `--port` (default: 8080 or next available open port) and `--no-browser` flags.
- Built automatic open port resolver (`findAvailablePort`) scanning from the base port to avoid port conflicts.
- Built cross-platform browser opener (`browserCommand` / `openBrowser`) targeting Linux (`xdg-open`), macOS (`open`), and Windows (`rundll32`).
- Connected graceful shutdown via `signal.NotifyContext` (SIGINT/SIGTERM) into `ui.Server`.
- Updated `skills/sage/SKILL.md`, `AGENTS.md`, `cmd/sage/cli/init.go`, `README.md`, `docs/SKILLS_SETUP.md`, `docs/CRON_WORKFLOW.md`, `docs/index.html`, `TERMS.md`, and `PRIVACY.md` removing deprecated commands and documenting `sage ui`.
- Comprehensive TDD unit and integration test coverage across all seams in `cmd/sage/cli/ui_test.go` and `root_test.go`.
