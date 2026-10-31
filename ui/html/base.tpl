<!DOCTYPE html>
<html lang="en" class="h-full">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Sage Architecture Visualizer</title>

  <!-- Local Tailwind CSS stylesheet -->
  <link rel="stylesheet" href="/static/css/tailwind.min.css">

  <style>
    /* =========================================================================
       THEME TOKENS (Light / Dark Mode)
       TWEAK HERE: Adjust these CSS variable hex codes to change palette colors.
       ========================================================================= */
    :root {
      --bg: #fbf9f4;
      --surface: #ffffff;
      --ink: #2b2a33;
      --muted: #6c6a78;
      --line: #d9d4c7;
      --accent: #10b981;
      color-scheme: light;
    }

    @media (prefers-color-scheme: dark) {
      :root,
      :root:not([data-theme="light"]) {
        --bg: #17171c;
        --surface: #202027;
        --ink: #ececf1;
        --muted: #9a98a8;
        --line: #3a3a46;
        --accent: #10b981;
        color-scheme: dark;
      }
    }

    :root[data-theme="dark"] {
      --bg: #17171c;
      --surface: #202027;
      --ink: #ececf1;
      --muted: #9a98a8;
      --line: #3a3a46;
      --accent: #10b981;
      color-scheme: dark;
    }

    :root[data-theme="light"] {
      --bg: #fbf9f4;
      --surface: #ffffff;
      --ink: #2b2a33;
      --muted: #6c6a78;
      --line: #d9d4c7;
      --accent: #10b981;
      color-scheme: light;
    }

    @font-face {
      font-family: 'Caveat';
      src: url('/static/fonts/Caveat-Regular.ttf') format('truetype');
    }

    body {
      font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
      background: var(--bg);
      color: var(--ink);
    }

    /* Action buttons styled with theme tokens */
    .btn-action {
      background: var(--surface);
      border: 1px solid var(--line);
      color: var(--ink);
    }

    /* Ensure rendered D2 SVG scales and fits stage */
    #diagram-stage svg {
      width: 100%;
      height: 100%;
      max-width: none;
      display: block;
      cursor: grab;
    }

    #diagram-stage svg:active {
      cursor: grabbing;
    }

    /* Interactive highlighting styles */
    .d2-dimmed {
      opacity: 0.22 !important;
      transition: opacity 0.2s ease-in-out;
    }

    .d2-highlighted {
      opacity: 1.0 !important;
      filter: drop-shadow(0 0 6px var(--accent));
      transition: filter 0.2s ease-in-out;
    }
  </style>
</head>
<body class="h-full flex flex-col overflow-hidden">

  <!-- Header Navigation Toolbar -->
  <header class="flex items-center justify-between px-4 py-2 shadow-sm z-10 select-none" style="background: var(--surface); border-bottom: 1px solid var(--line);">
    <div class="flex items-center space-x-3">
      <!-- Sage Brand -->
      <div class="flex items-center space-x-2">
        <svg class="w-5 h-5 text-emerald-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253"></path>
        </svg>
        <span class="text-xl font-bold tracking-tight text-emerald-500">Sage</span>
      </div>

      <!-- Active File Badge -->
      <span id="active-file" class="text-xs px-2 py-0.5 rounded font-mono" style="background: var(--bg); border: 1px solid var(--line); color: var(--muted);">
        {{ if .ActiveFile }}{{ .ActiveFile }}{{ else }}SAGE.md{{ end }}
      </span>
    </div>

    <!-- Live Indicator & Controls -->
    <div class="flex items-center space-x-4">
      <div id="live-indicator" class="flex items-center space-x-1.5 text-xs" style="color: var(--muted);">
        <span id="live-dot" class="inline-block w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></span>
        <span id="live-text">Live</span>
      </div>

      <div class="flex items-center space-x-1 pl-3" style="border-left: 1px solid var(--line);">
        <button id="btn-theme-toggle" title="Toggle Dark/Light Mode" class="btn-action px-2 py-1 text-xs rounded transition-opacity hover:opacity-80">🌓 Theme</button>
        <button id="btn-zoom-in" title="Zoom In (+)" class="btn-action px-2 py-1 text-xs rounded transition-opacity hover:opacity-80">Zoom In</button>
        <button id="btn-zoom-out" title="Zoom Out (-)" class="btn-action px-2 py-1 text-xs rounded transition-opacity hover:opacity-80">Zoom Out</button>
        <button id="btn-reset-view" title="Reset View (⟲)" class="btn-action px-2 py-1 text-xs rounded transition-opacity hover:opacity-80">Reset View</button>
        <button id="btn-export-svg" title="Export SVG" class="btn-action px-2 py-1 text-xs rounded transition-opacity hover:opacity-80">Export SVG</button>
      </div>
    </div>
  </header>

  <!-- Collapsible Diagnostic / Warning Banner -->
  {{ $hasWarnings := .Warnings }}
  <details id="warning-banner" class="px-4 py-2 bg-amber-900/40 border-b border-amber-600/40 text-amber-200 text-xs z-10 cursor-pointer {{ if not $hasWarnings }}hidden{{ end }}" {{ if $hasWarnings }}open{{ end }}>
    <summary class="font-semibold uppercase tracking-wider text-[10px] bg-amber-800 text-amber-100 px-1.5 py-0.5 rounded inline-block select-none">
      ⚠️ Diagnostics (Click to collapse/expand)
    </summary>
    <div id="warning-message" class="mt-2 pl-2 space-y-0.5 cursor-auto">
      {{ range .Warnings }}
      <div>{{ . }}</div>
      {{ end }}
    </div>
  </details>

  <!-- Main Visualizer Diagram Stage -->
  <main class="flex-1 relative overflow-hidden">
    <div id="diagram-stage" class="w-full h-full flex items-center justify-center">
      {{ .DiagramSVG }}
    </div>
  </main>

  <!-- Modular Scripts -->
  <script src="/static/js/panzoom.tpl"></script>
  <script src="/static/js/app.tpl"></script>
  <script src="/api/wait"></script>
</body>
</html>
