# SAGE Architecture Map

```d2
direction: right

cli: "CLI Entrypoint (cmd/sage)" {
  user: "Terminal Developer" {
    shape: circle
    style.fill: "#14532d"
    style.stroke: "#4ade80"
  }
  main: "Main Entrypoint (main.go)" {
    shape: rectangle
    style.fill: "#1e3a8a"
    style.stroke: "#60a5fa"
  }
  cli_root: "Cobra Command Dispatcher" {
    shape: diamond
    style.fill: "#713f12"
    style.stroke: "#facc15"
  }
  cmd_init: "Init Command (init.go)" {
    shape: rectangle
    style.fill: "#1e3a8a"
    style.stroke: "#60a5fa"
  }
  cmd_ui: "UI Command & Workflow (ui.go)" {
    shape: rectangle
    style.fill: "#1e3a8a"
    style.stroke: "#60a5fa"
  }
}

server: "Web Visualizer Server (internal/ui)" {
  ui_server: "HTTP Server & Router (server.go)" {
    shape: rectangle
    style.fill: "#1e3a8a"
    style.stroke: "#60a5fa"
  }
  sse_watcher: "File Watcher & SSE Live Reload" {
    shape: hexagon
    style.fill: "#7c2d12"
    style.stroke: "#fb923c"
  }
}

d2_engine: "D2 Engine (internal/d2)" {
  d2_compiler: "D2 Compiler (d2.go)" {
    shape: rectangle
    style.fill: "#1e3a8a"
    style.stroke: "#60a5fa"
  }
  d2_svg: "D2 SVG Renderer" {
    shape: cylinder
    style.fill: "#581c87"
    style.stroke: "#c084fc"
  }
}

frontend: "Frontend & Canvas (ui)" {
  browser: "Web Browser Client" {
    shape: circle
    style.fill: "#14532d"
    style.stroke: "#4ade80"
  }
  embedded_assets: "Embedded FS & HTML Template" {
    shape: rectangle
    style.fill: "#1e3a8a"
    style.stroke: "#60a5fa"
  }
  panzoom: "SVG Pan & Zoom (panzoom.tpl)" {
    shape: rectangle
    style.fill: "#334155"
    style.stroke: "#94a3b8"
  }
}

storage: "Architecture Data (SAGE.md)" {
  sage_file: "SAGE.md Spec" {
    shape: cylinder
    style.fill: "#581c87"
    style.stroke: "#c084fc"
  }
}

distribution: "Skills & Distribution (skills/sage)" {
  agent_skill: "Agent Skill & Syntax Spec" {
    shape: rectangle
    style.fill: "#334155"
    style.stroke: "#94a3b8"
  }
  installer: "Installer Script (assets/install.sh)" {
    shape: rectangle
    style.fill: "#334155"
    style.stroke: "#94a3b8"
  }
}

# Connections: Execution & CLI
cli.user -> cli.main: "Runs 'sage' in terminal"
cli.main -> cli.cli_root: "cli.Execute()"
cli.cli_root -> cli.cmd_init: "Dispatches 'sage init'"
cli.cli_root -> cli.cmd_ui: "Dispatches default / 'sage ui'"
cli.cmd_init -> storage.sage_file: "Creates starter template"

# Connections: Web Server & Browser
cli.cmd_ui -> server.ui_server: "Starts HTTP server"
cli.cmd_ui -> frontend.browser: "Launches web browser"
frontend.browser -> server.ui_server: "Requests UI & /api/diagram"
frontend.browser <-> server.sse_watcher: "EventSource stream /api/events"
frontend.browser -> frontend.panzoom: "Panning and zooming SVG"

# Connections: Parsing & Templates
server.ui_server -> frontend.embedded_assets: "Serves base.tpl & static assets"
server.ui_server -> d2_engine.d2_compiler: "Invokes ParseFile()"
d2_engine.d2_compiler -> storage.sage_file: "Reads D2 statements"
d2_engine.d2_compiler -> d2_engine.d2_svg: "Generates SVG with Dagre layout"
server.ui_server -> frontend.browser: "Transmits inlined D2 SVG"

# Connections: File Watcher & Distribution
server.sse_watcher -> storage.sage_file: "Polls os.Stat() mod time"
distribution.agent_skill -> storage.sage_file: "Maintains D2 architecture"
distribution.installer -> cli.user: "Installs binary & agent skills"
```
