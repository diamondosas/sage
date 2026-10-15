# Ticket 01: Purge Deprecated Commands and MCP Server

Status: resolved
Blocked by: none

## What it delivers

Removes deprecated human text inspection commands (`sage who`, `sage why`) and the unnecessary `sage mcp` server from the codebase:
- Remove `cmd/sage/cli/who.go`, `cmd/sage/cli/why.go`, and `cmd/sage/cli/mcp.go`.
- Remove `internal/mcp/` directory.
- Update `cmd/sage/cli/root.go` and `cmd/sage/cli/root_test.go` to remove command registrations.
- Clean up `go.mod` / `go.sum` of unused MCP dependencies if any.
- Keep `QueryOwner` and `QueryWhy` in `internal/graph/sqlite.go` as underlying query engines for the new interactive UI.

## Comments

- Removed `cmd/sage/cli/who.go`, `cmd/sage/cli/why.go`, and `cmd/sage/cli/mcp.go`.
- Removed `internal/mcp/` directory.
- Updated `cmd/sage/cli/root.go` and `cmd/sage/cli/root_test.go` to purge command registrations and assert they are no longer registered.
- Cleaned up `go.mod` and `go.sum` using `go mod tidy`.
- Preserved `QueryOwner` and `QueryWhy` in `internal/graph/sqlite.go`.

