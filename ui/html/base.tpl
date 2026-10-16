<!DOCTYPE html>
<html lang="en" class="h-full bg-slate-900 text-slate-100">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Sage Architecture Visualizer</title>

  <!-- Local Tailwind CSS stylesheet embedded in binary.
       CUSTOMIZATION: Change the path if serving behind a custom prefix (e.g. /app/static/css/...) -->
  <link rel="stylesheet" href="/static/css/tailwind.min.css">

  <style>
    /* CUSTOMIZATION: Change font family to match your preferred typography */
    body { font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; }
    /* Disable default user select on interactive canvas during pan/drag */
    #sage-canvas { user-select: none; -webkit-user-select: none; }
  </style>
</head>
<!-- CUSTOMIZATION: Change background (e.g. bg-gray-50 for light mode, bg-slate-900 for dark mode) -->
<body class="h-full flex flex-col overflow-hidden bg-slate-900 text-slate-100">

  <!-- Top Navigation Header:
       CUSTOMIZATION: Adjust height via py-2/py-3, header background via bg-slate-800, and border color -->
  <header class="flex items-center justify-between px-4 py-2 bg-slate-800 border-b border-slate-700 shadow-sm z-10">
    <div class="flex items-center space-x-3">
      <!-- Sage Logo & Brand Title:
           CUSTOMIZATION: Change logo color via text-emerald-400 or replace SVG with custom branding -->
      <div class="flex items-center space-x-2">
        <svg class="w-5 h-5 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253"></path>
        </svg>
        <span class="text-xl font-bold tracking-tight text-emerald-400">Sage</span>
      </div>

      <!-- Active File Name Badge -->
      <span id="active-file" class="text-xs px-2 py-0.5 rounded bg-slate-700 text-slate-300 font-mono">
        {{ if .ActiveFile }}{{ .ActiveFile }}{{ else }}SAGE.md{{ end }}
      </span>
    </div>

    <!-- Live Connection Status Badge and Action Controls -->
    <div class="flex items-center space-x-4">
      <!-- Live Indicator (Connected via Server-Sent Events) -->
      <div id="live-indicator" class="flex items-center space-x-1.5 text-xs text-slate-400">
        <span id="live-dot" class="inline-block w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span>
        <span id="live-text">Live</span>
      </div>

      <!-- Quick Action Controls (Zoom In, Zoom Out, Reset View, Export SVG):
           CUSTOMIZATION: Adjust button styling, border, or hover colors (e.g. hover:bg-slate-700) -->
      <div class="flex items-center space-x-1 border-l border-slate-700 pl-3">
        <button id="btn-zoom-in" title="Zoom In" class="px-2 py-1 text-xs rounded hover:bg-slate-700 text-slate-300">Zoom In</button>
        <button id="btn-zoom-out" title="Zoom Out" class="px-2 py-1 text-xs rounded hover:bg-slate-700 text-slate-300">Zoom Out</button>
        <button id="btn-reset-view" title="Reset View" class="px-2 py-1 text-xs rounded hover:bg-slate-700 text-slate-300">Reset View</button>
        <button id="btn-export-svg" title="Export SVG" class="px-2 py-1 text-xs rounded hover:bg-slate-700 text-slate-300">Export SVG</button>
      </div>
    </div>
  </header>

  <!-- Collapsible Notification / Warning Banner:
       Displays syntax warnings or fallback nodes if Graph.Warnings (or .Warnings) is non-empty.
       CUSTOMIZATION: Change alert colors (bg-amber-900/40, text-amber-200) for different alert styles -->
  {{ $warnings := or .Warnings (and .Graph .Graph.Warnings) }}
  {{ if $warnings }}
  <details id="warning-banner" open class="px-4 py-2 bg-amber-900/40 border-b border-amber-600/40 text-amber-200 text-xs z-10 cursor-pointer">
    <summary class="font-semibold uppercase tracking-wider text-[10px] bg-amber-800 text-amber-100 px-1.5 py-0.5 rounded inline-block select-none">
      ⚠️ Warnings (Click to collapse/expand)
    </summary>
    <div id="warning-message" class="mt-2 pl-2 space-y-0.5 cursor-auto">
      {{ range $warnings }}
      <div>{{ . }}</div>
      {{ end }}
    </div>
  </details>
  {{ end }}

  <!-- Main Visualizer SVG Canvas Container:
       CUSTOMIZATION: Change canvas background color via bg-slate-900 -->
  <main class="flex-1 relative overflow-hidden bg-slate-900">
    <svg id="sage-canvas" class="w-full h-full cursor-grab active:cursor-grabbing">
      <defs>
        <!-- Arrowhead markers for directed edges rendered by app.js -->
        <marker id="arrowhead" viewBox="0 0 10 10" refX="8" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse">
          <path d="M 0 1 L 8 5 L 0 9 z" fill="#94a3b8" />
        </marker>
        <marker id="arrowhead-start" viewBox="0 0 10 10" refX="2" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse">
          <path d="M 8 1 L 0 5 L 8 9 z" fill="#94a3b8" />
        </marker>
      </defs>
      <!-- Viewport group for pan and zoom transformations -->
      <g id="viewport">
        <!-- Dynamic graph nodes and edges will be inserted dynamically here -->
        <g id="edges-layer"></g>
        <g id="nodes-layer"></g>
      </g>
    </svg>
  </main>

  <!-- Initial Graph Data from Go template:
       Data model is serialized to window.__INITIAL_GRAPH__ for client-side Dagre layout. -->
  <script>
    window.__INITIAL_GRAPH__ = {{ .GraphJSON }};
  </script>

  <!-- Standalone vendored dependencies (no NPM required) -->
  <script src="/static/js/libs/rough.js"></script>
  <script src="/static/js/libs/dagre.min.js"></script>

  <!-- Client Application Script:
       CUSTOMIZATION: Modify ui/js/app.js to change layout spacing, colors, or physics -->
  <script src="/static/js/app.js"></script>
</body>
</html>
