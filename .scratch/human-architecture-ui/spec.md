# Spec: Human-Facing Non-Slop Architecture Visualizer

Status: ready-for-agent

## Problem Statement

Human engineers, contributors, and maintainers looking at a codebase struggle to understand how files connect, why certain architectural boundaries exist, and where architectural responsibilities lie. Existing automated graph diagrams produce dense, unreadable hairballs ("slop") that cause cognitive overload. Terminal commands (`sage who`, `sage why`) provide text outputs, but lack a human-friendly visual overview of architectural clusters and dependencies.

## Solution

Provide a local, zero-friction interactive web visualizer command (`sage ui`). It embeds a high-end, clean architectural visualizer directly within the single Go binary using embedded static assets. Instead of a chaotic full-mesh file graph, the UI groups the codebase into high-level architectural domain clusters (e.g., CLI, AI Synthesizer, Git Extractor, Graph Storage, MCP Protocol) and lets users click into any cluster to explore individual files, inspect their architectural role, examine dependencies/exports, and trace decisions/why lineage.

## User Stories

1. As a new contributor, I want to run `sage ui`, so that my browser opens to an immediate visual overview of the system architecture without installing npm packages or third-party web runtimes.
2. As a software architect, I want files grouped into high-level architectural domain clusters, so that I see clean conceptual boundaries rather than an overwhelming graph hairball.
3. As a developer, I want to click on a domain cluster, so that I can inspect all files belonging to that architectural role and their inter-cluster connections.
4. As a code reviewer, I want to click any file node in the graph, so that I can view its purpose, exported symbols, imports, formal CODEOWNER, and recent expert contributors in a side inspector.
5. As a developer investigating a legacy module, I want to view the "Why Lineage" in the UI inspector, so that I understand PR decisions and commits that shaped the component without running manual terminal queries.
6. As a CI/CD operator or script, I want to pass `--no-browser` and `--port`, so that the server can run headless or bind to a specific port for remote access or testing.
7. As an AI agent working in the repo, I want access to the packaged visualizer skill and endpoints, so that I can audit structural debt and verify architecture impact after code refactoring.

## Implementation Decisions

- **Single Binary Web Serving**: The visualization assets (HTML, CSS, JavaScript) will be embedded directly into the Go binary using `embed.FS`. The web server requires zero external processes (no node, python, or external web server).
- **Embedded API Server**: The server provides JSON REST endpoints:
  - `GET /api/graph`: Returns structured domain clusters, nodes, and inter-cluster connection edges extracted from the SQLite store.
  - `GET /api/file`: Returns deep inspection details (role, summary, exports, dependencies, owner, experts, decision lineage) for a requested file path.
- **Visual Design Standard**: Strict non-slop aesthetic: calm dark/light modern typography, clean geometric cluster cards/nodes, smooth physics layout, and a distraction-free inspection drawer.
- **Browser Automation**: Automatically launches the system default browser upon running `sage ui` unless `--no-browser` is specified.
- **Agent Skill Integration**: The repository skills include instructions for agents to inspect architecture clusters and use the visualizer to avoid adding cross-module structural debt.

## Testing Decisions

- **Testing Philosophy**: Test external behavior at the HTTP boundary rather than internal canvas rendering or private structs.
- **Tested Components**:
  - The embedded HTTP server handler and API endpoints.
  - Graph serialization: validating cluster extraction and node mapping against a test SQLite database.
  - CLI argument parsing for port and browser flags.
- **Prior Art**: Follow the patterns established in `internal/ai/ai_test.go` and `internal/graph/sqlite_test.go`, which spin up in-memory or temporary SQLite test instances.

## Out of Scope

- Hosted cloud deployment or multi-user authentication (this is a local developer tool).
- Graph editing / modifying architecture from the web interface.
- 3D rendering engines (keep it lightweight 2D canvas/SVG with zero heavy external runtime dependencies).

## Further Notes

- Future iterations may add static export (`sage export --html`) for deployment on GitHub Pages documentation.
