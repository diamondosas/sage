# SAGE Architecture Language Specification (`language-syntax.md`)

The standard syntax for `SAGE.md` is standard [D2 language](https://d2lang.com) markup. AI coding agents use this specification to generate and incrementally maintain codebase architecture diagrams rendered by the `sage` visualizer.

---

## 1. Golden Rule: No Assumptions
> **CRITICAL**: The AI must never assume facts about the codebase. Every node, arrow, container, and file path must be verified against actual repository files, imports, and `git status`/`git diff`. Never speculate or invent phantom modules.

---

## 2. File Format & Code Fences
An architecture diagram file (`SAGE.md`) encloses D2 markup within a standard ```` ```d2 ```` block:

````markdown
# Architecture Map

```d2
direction: right
# CHANGE DIRECTION: Replace "right" with "down" to render a vertical top-to-bottom layout
```
````

*(Note: Sage also provides seamless backward compatibility for legacy ```` ```sage ```` blocks and raw un-fenced D2 input).*

---

## 3. Containers (Codebase Directory Clustering)
Group related components into architectural containers that map directly to codebase folders or domains using an identifier followed by a quoted label:

```d2
direction: right

storage: "Storage Layer (internal/storage)" {
  # CHANGE CONTAINER: Update label and verified codebase directory in quotes
  db: "PostgreSQL DB" {
    shape: cylinder
    style.fill: "#581c87"
    style.stroke: "#c084fc"
  }
}
```

- **Container Definition**: `identifier: "Human-Readable Label (path/to/dir)" { ... }`.
- **Node Identifier**: Clean alphanumeric identifier without spaces (e.g., `db`, `api_server`, `router`).

---

## 4. Component Shapes Cheat Sheet
Define architectural nodes using standard D2 shapes:

| Component Type | D2 Syntax | Shape Value | Semantic Role |
| :--- | :--- | :--- | :--- |
| **Service / Module** | `service: "API Service" { shape: rectangle }` | `rectangle` | Core services, packages, HTTP handlers, controllers |
| **User / Consumer** | `user: "Web Browser" { shape: circle }` | `circle` | Terminal developers, human users, external clients |
| **Decision / Gateway** | `auth: "Auth Gate" { shape: diamond }` | `diamond` | Routers, API gateways, decision points, evaluators |
| **Worker / Queue** | `worker: "Worker Queue" { shape: queue }` | `queue` | Background jobs, queue consumers, async event handlers |
| **Database / Store** | `db: "Main Database" { shape: cylinder }` | `cylinder` | Persistent databases, file systems, disk caches |
| **Event Watcher** | `watcher: "File Watcher" { shape: hexagon }` | `hexagon` | File watchers, monitors, triggers |

*(Note: D2 does NOT support `shape: triangle` for nodes—triangle is reserved for arrowheads).*

---

## 5. Palette Color Styling
Sage visualizer comes pre-calibrated with 7 dark-slate architectural color classes. Attach color styling to any node using `style.fill` and `style.stroke`:

- **Blue** (`fill: "#1e3a8a"`, `stroke: "#60a5fa"`) - Core services, application packages, and controllers.
- **Green** (`fill: "#14532d"`, `stroke: "#4ade80"`) - Entrypoints, human users, clients, and healthy interfaces.
- **Yellow** (`fill: "#713f12"`, `stroke: "#facc15"`) - Decision gates, auth verifiers, and command dispatchers.
- **Purple** (`fill: "#581c87"`, `stroke: "#c084fc"`) - Persistent storage, databases, and graph AST models.
- **Orange** (`fill: "#7c2d12"`, `stroke: "#fb923c"`) - Background workers, file watchers, and asynchronous queue consumers.
- **Red** (`fill: "#7f1d1d"`, `stroke: "#f87171"`) - External untrusted APIs, error boundaries, and monitor alerts.
- **Gray** (`fill: "#334155"`, `stroke: "#94a3b8"`) - Shared utilities, helper libraries, and neutral fallback modules.

### Example Node with Styling

```d2
app: "Application Service" {
  shape: rectangle
  style.fill: "#1e3a8a"
  style.stroke: "#60a5fa"
}
```

---

## 6. Dependency Arrows & Connections
Connect components with explicit dependency edges and clear action descriptions. Use container-scoped identifiers (`container.node`) when referencing elements across containers:

- **Directed Call / Dependency**:
  `cli.user -> cli.main: "Runs CLI command"`
- **Cross-container Call**:
  `cli.router -> server.http_srv: "Starts local server"`
- **Bidirectional Synchronization**:
  `server.http_srv <-> server.watcher: "Live sync stream"`

---

## 7. Incremental Maintenance Protocol for AI Agents
When modifying or updating an existing architecture diagram:
1. **Never wipe or regenerate the entire file**: AI models must respect established diagrams and perform minimal, surgical diffs.
2. **Localize changes**:
   - Run `git status` and `git diff` to identify touched modules.
   - Find the existing container corresponding to the touched codebase path.
   - Surgically add or update only the affected nodes and connection arrows.
3. **Preserve surrounding context**: Retain existing nodes, connections, and container layouts for untouched modules.
4. **Explain simply**: Write component labels and edge descriptions as if teaching a child—straightforward, concise, and focused on structure.

---

## 8. Complete Working Example

```d2
direction: right

cli: "CLI Layer (cmd/sage)" {
  user: "Developer User" {
    shape: circle
    style.fill: "#14532d"
    style.stroke: "#4ade80"
  }
  app: "Cobra CLI" {
    shape: rectangle
    style.fill: "#1e3a8a"
    style.stroke: "#60a5fa"
  }
  router: "Command Router" {
    shape: diamond
    style.fill: "#713f12"
    style.stroke: "#facc15"
  }
}

server: "Web Server (internal/ui)" {
  http_srv: "HTTP Server" {
    shape: rectangle
    style.fill: "#1e3a8a"
    style.stroke: "#60a5fa"
  }
  watcher: "File Watcher" {
    shape: hexagon
    style.fill: "#7c2d12"
    style.stroke: "#fb923c"
  }
}

engine: "D2 Engine (internal/d2)" {
  compiler: "D2 Compiler" {
    shape: rectangle
    style.fill: "#1e3a8a"
    style.stroke: "#60a5fa"
  }
}

# Connections
cli.user -> cli.app: "Runs CLI command"
cli.app -> cli.router: "Dispatches"
cli.router -> server.http_srv: "Starts local server"
server.http_srv -> engine.compiler: "Parses SAGE.md"
server.watcher -> server.http_srv: "Triggers SSE reload"
```
