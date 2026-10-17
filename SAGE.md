# SAGE Architecture Map

```sage
// group: CLI Entrypoint
// path: cmd/sage
circle user "Terminal Developer" [green]
rect main "Main Entrypoint (main.go)" [blue]
diamond cli_root "Cobra Command Dispatcher" [yellow]
rect cmd_init "Init Command (init.go)" [blue]
rect cmd_ui "UI Command & Workflow (ui.go)" [blue]

// group: Web Visualizer Server
// path: internal/ui
rect ui_server "HTTP Server & Router (server.go)" [blue]
triangle sse_watcher "File Watcher & SSE Live Reload" [orange]

// group: DSL Engine
// path: internal/parser
rect dsl_parser "DSL Parser (parser.go)" [blue]
cylinder graph_ast "Graph AST & Types (types.go)" [purple]

// group: Frontend & Canvas
// path: ui
circle browser "Web Browser Client" [green]
rect embedded_assets "Embedded FS & HTML Template" [blue]
rect canvas_app "Canvas Renderer (Rough.js & Dagre)" [yellow]

// group: Architecture Data
// path: SAGE.md
cylinder sage_file "SAGE.md Spec" [purple]

// group: Skills & Distribution
// path: skills/sage
rect agent_skill "Agent Skill & Syntax Spec" [gray]
rect installer "Installer Script (assets/install.sh)" [gray]

// Connections: Execution & CLI
user -> main "Runs 'sage' in terminal"
main -> cli_root "cli.Execute()"
cli_root -> cmd_init "Dispatches 'sage init'"
cli_root -> cmd_ui "Dispatches default / 'sage ui'"
cmd_init -> sage_file "Creates starter template"

// Connections: Web Server & Browser
cmd_ui -> ui_server "Starts HTTP server"
cmd_ui -> browser "Launches web browser"
browser -> ui_server "Requests UI & /api/graph"
browser <-> sse_watcher "EventSource stream /api/events"
browser -> canvas_app "Renders interactive diagram"

// Connections: Parsing & Templates
ui_server -> embedded_assets "Serves base.tpl & static assets"
ui_server -> dsl_parser "Invokes ParseFile()"
dsl_parser -> sage_file "Reads DSL statements"
dsl_parser -> graph_ast "Generates Nodes, Edges, & Warnings"
ui_server -> browser "Transmits Graph JSON"

// Connections: File Watcher & Distribution
sse_watcher -> sage_file "Polls os.Stat() mod time"
agent_skill -> sage_file "Maintains DSL architecture"
installer -> user "Installs binary & agent skills"
```
