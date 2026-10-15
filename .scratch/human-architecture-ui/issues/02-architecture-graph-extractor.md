# Ticket 02: Architecture Domain Cluster & Dependency Extractor

Status: resolved
Blocked by: 01

## What it delivers

Implements domain cluster aggregation in `internal/graph`:
- Groups stored file records into coarse-grained architectural clusters (e.g. CLI, AI Synthesizer, Git Extractor, Graph Engine, Configuration, Testing).
- Extracts inter-cluster dependency links based on parsed imports and file relations.
- Provides `QueryArchitectureGraph() (*ArchitectureGraph, error)` in `internal/graph/sqlite.go`.
- Unit tests in `internal/graph/sqlite_test.go` verifying accurate clustering, node generation, and dependency edge formation without cyclic hairballs.

## Comments

- Added domain cluster models, constants, and classifications in `internal/graph/architecture.go`.
- Implemented `BuildArchitectureGraph` to group files into coarse-grained clusters and calculate inter-cluster dependency edges, suppressing self-loops and redundant cyclic edges.
- Implemented `Engine.QueryArchitectureGraph()` in `internal/graph/sqlite.go` to load all file records from SQLite and construct the architectural graph.
- Added comprehensive unit tests in `internal/graph/architecture_test.go` and `internal/graph/sqlite_test.go` verifying empty-state handling, accurate clustering, and weighted directed edges.

