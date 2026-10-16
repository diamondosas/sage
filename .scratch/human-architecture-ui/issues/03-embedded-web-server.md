# Ticket 03: Embedded Web Server & REST API Endpoints

Status: resolved
Blocked by: 02

## What it delivers

Implements a standalone HTTP server in `internal/ui`:
- Embeds frontend static assets using Go `embed.FS`.
- Exposes REST API endpoints:
  - `GET /api/graph`: Returns domain clusters, nodes, and dependency edges.
  - `GET /api/file?path=<path>`: Returns detailed inspection data (role, summary, exports, dependencies, formal owner, recent experts, and decision lineage via `QueryWhy`).
- Graceful shutdown and port binding configuration (`--port`).
- HTTP integration tests verifying endpoint responses against a test database.

## Comments

- Implemented embedded web server in `internal/ui/server.go` using Go `embed.FS` for zero-friction single-binary distribution.
- Added `GET /api/graph` endpoint connecting to `Engine.QueryArchitectureGraph()`.
- Added `GET /api/file?path=<path>` endpoint delivering rich file inspection data (role, purpose summary, exported symbols, imports, formal CODEOWNER, recent expert contributors, and decision lineage).
- Supported port binding configuration and context-driven graceful shutdown with connection draining.
- Wrote thorough HTTP integration tests in `internal/ui/server_test.go` verifying endpoint responses, error conditions, and server lifecycle.

