<!DOCTYPE html>
<html lang="en" class="h-full">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Sage Architecture Visualizer</title>

  <!-- Local Tailwind CSS stylesheet embedded in binary.
       CUSTOMIZATION: Change the path if serving behind a custom prefix (e.g. /app/static/css/...) -->
  <link rel="stylesheet" href="/static/css/tailwind.min.css">

  <style>
    /* Warm paper & dark mode semantic tokens */
    /* TWEAK HERE: Modify light mode palette hex values */
    :root {
      --bg: #fbf9f4;
      --surface: #ffffff;
      --ink: #2b2a33;
      --muted: #6c6a78;
      --line: #d9d4c7;
      --accent: #e4572e;
      --node: #bcd4ff;
      --node2: #ffe3a3;
      --note: #fff3b0;
      color-scheme: light;
    }

    /* TWEAK HERE: Modify dark mode palette hex values */
    @media (prefers-color-scheme: dark) {
      :root:not([data-theme="light"]) {
        --bg: #17171c;
        --surface: #202027;
        --ink: #ececf1;
        --muted: #9a98a8;
        --line: #3a3a46;
        --accent: #ff7a59;
        --node: #3f5f94;
        --node2: #8a6a22;
        --note: #5c5320;
        color-scheme: dark;
      }
    }

    :root[data-theme="dark"] {
      --bg: #17171c;
      --surface: #202027;
      --ink: #ececf1;
      --muted: #9a98a8;
      --line: #3a3a46;
      --accent: #ff7a59;
      --node: #3f5f94;
      --node2: #8a6a22;
      --note: #5c5320;
      color-scheme: dark;
    }

    /* Local handwriting font definition for 100% offline diagram rendering */
    /* TWEAK HERE: Update font-family or swap font weight if needed */
    @font-face {
      font-family: 'Caveat';
      src: url('/static/fonts/Caveat-Regular.ttf') format('truetype');
    }

    /* CUSTOMIZATION: Change font family to match your preferred typography */
    body { font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; }
    /* Apply Caveat handwriting font to Mermaid diagram elements */
    .mermaid text, .mermaid tspan, #sage-canvas text, #diagram-stage text {
      font-family: 'Caveat', cursive, sans-serif;
      font-size: 20px;
    }
    /* Disable default user select on interactive canvas during pan/drag */
    #sage-canvas { user-select: none; -webkit-user-select: none; }
  </style>
</head>
<!-- CUSTOMIZATION: Canvas background and ink color adapt via CSS variables -->
<body class="h-full flex flex-col overflow-hidden" style="background: var(--bg); color: var(--ink);">

  <!-- Top Navigation Header:
       CUSTOMIZATION: Adjust height via py-2/py-3, header background via var(--surface), and border via var(--line) -->
  <header class="flex items-center justify-between px-4 py-2 shadow-sm z-10" style="background: var(--surface); border-bottom: 1px solid var(--line); color: var(--ink);">
    <div class="flex items-center space-x-3">
      <!-- Sage Logo & Brand Title:
           CUSTOMIZATION: Change logo color via text-emerald-500 or replace SVG with custom branding -->
      <div class="flex items-center space-x-2">
        <svg class="w-5 h-5 text-emerald-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253"></path>
        </svg>
        <span class="text-xl font-bold tracking-tight text-emerald-500">Sage</span>
      </div>

      <!-- Active File Name Badge -->
      <span id="active-file" class="text-xs px-2 py-0.5 rounded font-mono" style="background: var(--bg); border: 1px solid var(--line); color: var(--muted);">
        {{ if .ActiveFile }}{{ .ActiveFile }}{{ else }}SAGE.md{{ end }}
      </span>
    </div>

    <!-- Live Connection Status Badge and Action Controls -->
    <div class="flex items-center space-x-4">
      <!-- Live Indicator (Connected via Server-Sent Events) -->
      <div id="live-indicator" class="flex items-center space-x-1.5 text-xs" style="color: var(--muted);">
        <span id="live-dot" class="inline-block w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></span>
        <span id="live-text">Live</span>
      </div>

      <!-- Quick Action Controls (Zoom In, Zoom Out, Reset View, Export SVG):
           CUSTOMIZATION: Buttons styled with semantic tokens (var(--surface), var(--line), var(--ink)) -->
      <div class="flex items-center space-x-1 pl-3" style="border-left: 1px solid var(--line);">
        <button id="btn-zoom-in" title="Zoom In" class="px-2 py-1 text-xs rounded transition-opacity hover:opacity-80" style="background: var(--surface); border: 1px solid var(--line); color: var(--ink);">Zoom In</button>
        <button id="btn-zoom-out" title="Zoom Out" class="px-2 py-1 text-xs rounded transition-opacity hover:opacity-80" style="background: var(--surface); border: 1px solid var(--line); color: var(--ink);">Zoom Out</button>
        <button id="btn-reset-view" title="Reset View" class="px-2 py-1 text-xs rounded transition-opacity hover:opacity-80" style="background: var(--surface); border: 1px solid var(--line); color: var(--ink);">Reset View</button>
        <button id="btn-export-svg" title="Export SVG" class="px-2 py-1 text-xs rounded transition-opacity hover:opacity-80" style="background: var(--surface); border: 1px solid var(--line); color: var(--ink);">Export SVG</button>
      </div>
    </div>
  </header>

  <!-- Collapsible Notification / Warning Banner:
       Displays syntax warnings or fallback nodes if Graph.Warnings (or .Warnings) is non-empty.
       CUSTOMIZATION: Change alert colors (bg-amber-900/40, text-amber-200) for different alert styles -->
  {{ $warnings := or .Warnings (and .Graph .Graph.Warnings) }}
  <details id="warning-banner" class="px-4 py-2 bg-amber-900/40 border-b border-amber-600/40 text-amber-200 text-xs z-10 cursor-pointer {{ if not $warnings }}hidden{{ end }}" {{ if $warnings }}open{{ end }}>
    <summary class="font-semibold uppercase tracking-wider text-[10px] bg-amber-800 text-amber-100 px-1.5 py-0.5 rounded inline-block select-none">
      ⚠️ Warnings (Click to collapse/expand)
    </summary>
    <div id="warning-message" class="mt-2 pl-2 space-y-0.5 cursor-auto">
      {{ range $warnings }}
      <div>{{ . }}</div>
      {{ end }}
    </div>
  </details>

  <!-- Main Visualizer SVG Canvas Container:
       CUSTOMIZATION: Change canvas background color via var(--bg) -->
  <main class="flex-1 relative overflow-hidden" style="background: var(--bg);">
    <div id="diagram-stage" class="w-full h-full flex items-center justify-center">
      <svg id="sage-canvas" class="w-full h-full cursor-grab active:cursor-grabbing">
        <defs>
          <!-- Arrowhead markers for directed edges rendered by app.js -->
          <marker id="arrowhead" viewBox="0 0 10 10" refX="8" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse">
            <path d="M 0 1 L 8 5 L 0 9 z" fill="var(--ink, #2b2a33)" />
          </marker>
          <marker id="arrowhead-start" viewBox="0 0 10 10" refX="2" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse">
            <path d="M 8 1 L 0 5 L 8 9 z" fill="var(--ink, #2b2a33)" />
          </marker>
        </defs>
        <!-- Viewport group for pan and zoom transformations -->
        <g id="viewport">
          <!-- Dynamic graph nodes and edges will be inserted dynamically here -->
          <g id="edges-layer"></g>
          <g id="nodes-layer"></g>
        </g>
      </svg>
    </div>
  </main>

  <!-- Initial Graph Data from Go template:
       Data model is serialized to window.__INITIAL_GRAPH__ for client-side Dagre layout. -->
  <script>
    window.__INITIAL_GRAPH__ = {{ .GraphJSON }};
  </script>

  <!-- Standalone vendored dependencies (no NPM required) -->
  <script src="/static/js/libs/mermaid.min.js"></script>
  <script src="/static/js/libs/rough.js"></script>
  <script src="/static/js/libs/dagre.min.js"></script>

  <!-- Modular Visualizer Subsystems (zero-build, 100% offline) -->
  <script src="/static/js/sketchify.js"></script>
  <script src="/static/js/panzoom.js"></script>
  <script src="/static/js/annotations.js"></script>
  <script src="/static/js/export.js"></script>

  <!-- Client Application Script:
       CUSTOMIZATION: Modify ui/js/app.js to change layout spacing, colors, or physics -->
  <script src="/static/js/app.js"></script>
</body>
</html>
