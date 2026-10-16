# Sage Agent Instructions

This repository is indexed by Sage (SQLite: `.github/sage/sage.db`). When working in this codebase:
- Run 'sage ui --no-browser' to serve the architecture graph and query file details via REST (`/api/graph`, `/api/file?path=<path>`).
- Run 'sage overview' to see high-level repository knowledge statistics.
- Run 'sage query "<sql_or_search>"' to query or inspect stored records.

## Agent skills

### Issue tracker

Local markdown files in `.scratch/`. See `docs/agents/issue-tracker.md`.

### Triage labels

Five canonical triage labels (`needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`). See `docs/agents/triage-labels.md`.

### Domain docs

Single-context layout (`GLOSSARY.md` + `docs/adr/`). See `docs/agents/domain.md`.
