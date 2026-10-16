# SAGE Diagram Language Specification (`lang-docs.md`)

The SAGE DSL is a lightweight, line-oriented diagram specification language designed for AI agents to write and update architecture maps in `SAGE.md`, rendered by the `sage` web visualizer for human users.

---

## 1. Golden Rule: No Assumptions
> **CRITICAL**: The AI must never assume facts about the codebase. Every node, arrow, group, and file path must be verified against actual repository files, imports, and `git status`/`git diff`. Never speculate or invent phantom modules.

---

## 2. File Format
A diagram file (`SAGE.md`) can contain plain DSL statements or enclose them inside a markdown fence block:

````markdown
```sage
// SAGE statements go here
```
````

Lines are processed sequentially. Blank lines and comments are ignored by the parser.

---

## 3. Comments & Metadata
- **Single-line comments**: Begin with `//` or `#`.
- **Group metadata**: `// group: <group_name>` assigns all subsequent nodes to the specified architectural domain cluster until another group is declared.
- **Path metadata**: `// path: <relative/path>` maps subsequent nodes or groups to verified codebase directories or files.

```sage
// group: Storage
// path: internal/storage
```

---

## 4. Node Shapes
Define nodes with the syntax:
```sage
<shape> <id> "<label>" [color]
```

- `<shape>`: Supported shapes and aliases:
  - `rect` (alias: `box`) - Standard module, package, or service.
  - `circle` - Entry point, client, or external consumer.
  - `diamond` - Decision gate, router, or gateway.
  - `triangle` - Event trigger, worker, or queue consumer.
  - `cylinder` (alias: `db`) - Database, cache, or persistent state.
  *(Change the shape keyword to switch the rendered Rough.js geometry in the web visualizer)*
- `<id>`: Unique alphanumeric identifier (no spaces, e.g. `api_gateway`, `user_db`).
- `"<label>"`: Human-readable display label enclosed in double quotes. Explain simply as if teaching a child.
- `[color]`: *(Optional)* Bracketed color tag. Default is `gray` if omitted.
  - Supported palette tags: `blue`, `green`, `red`, `yellow`, `purple`, `orange`, `gray`.
  - *(Change `[color]` tag to adjust the node's accent fill and stroke in the visualizer)*

```sage
// Examples covering all supported shapes:
circle client "Web Browser" [green]             // circle: entrypoints / consumers
rect api "API Gateway" [blue]                   // rect (or box): services / components
diamond auth "Auth Verifier" [yellow]           // diamond: decision points / routers
triangle worker "Background Worker" [orange]    // triangle: workers / queue triggers
cylinder db "PostgreSQL Database" [purple]      // cylinder (or db): storage / databases

// Examples covering remaining palette colors:
rect monitor "System Monitor" [red]             // red: alerts / external APIs
rect util "Helper Utilities" [gray]             // gray: neutral / utilities (default if omitted)
box legacy "Legacy Module"                      // default color is gray when omitted
```

---

## 5. Connections (Edges)
Define connections between existing nodes:

- **Directed connection**: `<source_id> -> <target_id> "<label>"`
- **Bidirectional connection**: `<source_id> <-> <target_id> "<label>"`
- `"<label>"` is optional. Double quotes contain the human-readable action or protocol.
*(Change `->` to `<->` to indicate bidirectional synchronization or two-way request/response instead of one-way data flow)*

```sage
// Change arrow to <-> if WebSocket or bidirectional stream
client -> api "HTTPS / REST"

// Change label to clarify query type or protocol
api <-> db "SQL Queries"
```

---

## 6. Fault Tolerance & Fallbacks
- **Undeclared references**: If an edge refers to an ID that has not been explicitly declared as a node, the parser automatically creates a fallback `rect` node labeled with the ID and styled in `gray`.
- **Invalid lines**: Any line that cannot be parsed is added to the graph's non-blocking warnings list, visible in the UI banner.

---

## 7. Complete Example

```sage
// group: Frontend
// path: ui/
// Change [green] to [orange] to signal an untrusted external client
circle browser "Human User" [green]

// group: Backend
// path: cmd/sage
rect cli "Cobra CLI" [blue]
diamond router "Request Router" [yellow]

// group: Engine
// path: internal/parser
rect parser "DSL Parser" [blue]
cylinder cache "Graph State" [purple]

// Connections: Change targets or labels to reflect code imports
browser -> cli "Runs 'sage' in terminal"
cli -> router "Dispatches command"
router -> parser "Loads SAGE.md"
parser -> cache "Stores AST nodes & edges"
```
