/**
 * Sage Architecture Visualizer - Frontend Application
 *
 * High-performance, clean architectural diagram visualizer powered directly by Mermaid.js.
 * Utilizes Mermaid's native hand-drawn rendering mode (look: 'handDrawn') with
 * theme-aware palette injection, auto-framing, pan/zoom, and live reload.
 */

(function (global) {
  'use strict';

  // =========================================================================
  // VISUAL PARAMETERS & CONFIGURATION
  // TWEAK HERE: Adjust hand-drawn look, font family, size, or default seed.
  // =========================================================================

  const MERMAID_FONT_FAMILY = "'Caveat', cursive, sans-serif";
  const MERMAID_FONT_SIZE = "20px";

  const DEFAULT_OPTIONS = {
    look: 'handDrawn',      // 'handDrawn' for architectural sketch style, 'classic' for crisp geometric
    seed: 7,                // Seed for repeatable hand-drawn line wobbles
    font: MERMAID_FONT_FAMILY,
    fontSize: MERMAID_FONT_SIZE,
    securityLevel: 'loose'
  };

  // THEME COLOR PALETTE (Light / Dark Mode Tokens)
  // TWEAK HERE: Modify these hex values to customize theme colors
  const PALETTE = {
    light: {
      bg: '#fbf9f4',
      surface: '#ffffff',
      ink: '#2b2a33',
      muted: '#6c6a78',
      line: '#d9d4c7',
      accent: '#e4572e',
      node: '#bcd4ff',
      node2: '#ffe3a3',
      note: '#fff3b0'
    },
    dark: {
      bg: '#17171c',
      surface: '#202027',
      ink: '#ececf1',
      muted: '#9a98a8',
      line: '#3a3a46',
      accent: '#ff7a59',
      node: '#3f5f94',
      node2: '#8a6a22',
      note: '#5c5320'
    }
  };

  /**
   * Resolves the active color palette based on explicit overrides, data-theme, or OS preference.
   */
  function getActiveThemePalette(override) {
    if (PALETTE[override]) return PALETTE[override];
    if (typeof document !== 'undefined') {
      const themeAttr = document.documentElement.getAttribute('data-theme');
      if (PALETTE[themeAttr]) return PALETTE[themeAttr];
      if (typeof window !== 'undefined' && window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
        return PALETTE.dark;
      }
    }
    return PALETTE.light;
  }

  /**
   * Maps palette tokens into standard Mermaid themeVariables.
   */
  function getMermaidThemeVariables(palette = getActiveThemePalette(), options = {}) {
    const font = options.font || options.fontFamily || MERMAID_FONT_FAMILY;
    const fontSize = options.fontSize || MERMAID_FONT_SIZE;
    return {
      fontFamily: font,
      fontSize: fontSize,
      background: palette.bg,
      textColor: palette.ink,
      titleColor: palette.ink,
      primaryColor: palette.node,
      primaryTextColor: palette.ink,
      primaryBorderColor: palette.ink,
      lineColor: palette.ink,
      secondaryColor: palette.node2,
      tertiaryColor: palette.surface,
      clusterBkg: palette.node2,
      clusterBorder: palette.ink,
      edgeLabelBackground: palette.surface,
      nodeBorder: palette.ink,
      mainBkg: palette.node,
      labelTextColor: palette.ink,
      noteBkgColor: palette.note,
      noteTextColor: palette.ink,
      noteBorderColor: palette.ink,
      actorBkg: palette.node,
      actorBorder: palette.ink,
      actorTextColor: palette.ink,
      actorLineColor: palette.muted,
      signalColor: palette.ink,
      signalTextColor: palette.ink
    };
  }

  const MERMAID_THEME_VARIABLES = getMermaidThemeVariables(PALETTE.light);

  /**
   * Builds the complete configuration object for Mermaid.js.
   * Following the reference implementation in extras/Mermaid × Rough.js comprehensive example.html.
   *
   * @param {Object} [options] - Options including look ('handDrawn' | 'classic'), seed, colors, font, fontSize
   * @returns {Object} Mermaid initialization configuration
   */
  function mermaidConfig(options = {}) {
    const look = options.look || DEFAULT_OPTIONS.look;
    const seed = options.seed !== undefined ? options.seed : DEFAULT_OPTIONS.seed;
    const colors = options.colors || getActiveThemePalette(options.theme);
    const font = options.font || options.fontFamily || DEFAULT_OPTIONS.font;
    const fontSize = options.fontSize || DEFAULT_OPTIONS.fontSize;
    const securityLevel = options.securityLevel || DEFAULT_OPTIONS.securityLevel;

    return {
      startOnLoad: false,
      theme: 'base',
      securityLevel: securityLevel,
      look: look === 'handDrawn' ? 'handDrawn' : 'classic',
      handDrawnSeed: seed,
      fontFamily: font,
      fontSize: fontSize,
      themeVariables: getMermaidThemeVariables(colors, { font, fontSize })
    };
  }

  // =========================================================================
  // VIEWPORT, PAN & ZOOM STATE
  // =========================================================================

  const viewState = {
    x: 0,
    y: 0,
    scale: 1,
    minScale: 0.15,
    maxScale: 3.0
  };

  let activePanZoomController = null;
  let activeAnnotationsController = null;
  const persistedHighlights = new Set();

  let mermaidRenderCounter = 0;
  let lastMermaidSource = null;
  let lastDiagramOptions = null;

  // DOM Elements
  const getEl = id => typeof document !== 'undefined' ? document.getElementById(id) : null;
  const warningBanner = getEl('warning-banner');
  const warningMessage = getEl('warning-message');
  const liveDot = getEl('live-dot');
  const liveText = getEl('live-text');

  /**
   * Wraps the inner elements of an SVG diagram inside a dedicated <g id="viewport"> group.
   */
  function wrapInViewport(svgEl) {
    if (!svgEl) return null;
    let vp = svgEl.querySelector('#viewport');
    if (vp) return vp;

    vp = document.createElementNS('http://www.w3.org/2000/svg', 'g');
    vp.setAttribute('id', 'viewport');

    const children = Array.from(svgEl.childNodes);
    children.forEach(child => {
      const tag = (child.tagName || '').toLowerCase();
      if (tag !== 'defs' && tag !== 'style') {
        vp.appendChild(child);
      }
    });

    svgEl.appendChild(vp);
    return vp;
  }

  /**
   * Automatically calculates scale and offset to center the diagram comfortably in the viewport.
   */
  function autoFitDiagram(svgEl, viewportEl) {
    if (!svgEl || !viewportEl) return;
    try {
      const bbox = viewportEl.getBBox();
      if (!bbox || bbox.width <= 0 || bbox.height <= 0) return;

      const stage = svgEl.parentElement || document.getElementById('diagram-stage');
      const stageW = (stage && stage.clientWidth) || window.innerWidth || 800;
      const stageH = (stage && stage.clientHeight) || window.innerHeight || 600;
      const padding = 40;

      const scaleX = (stageW - 2 * padding) / bbox.width;
      const scaleY = (stageH - 2 * padding) / bbox.height;
      let scale = Math.min(scaleX, scaleY, 1.2);
      scale = Math.max(viewState.minScale, Math.min(viewState.maxScale, scale));

      const x = (stageW - bbox.width * scale) / 2 - bbox.x * scale;
      const y = (stageH - bbox.height * scale) / 2 - bbox.y * scale;

      viewState.x = x;
      viewState.y = y;
      viewState.scale = scale;
      updateTransform();
    } catch (e) {
      // getBBox may not be available in headless environments; safe to ignore
    }
  }

  /**
   * Ensures all cluster/subgraph groups have completely solid fills without hachure gaps.
   */
  function ensureClusterSolidFill(svgEl, customPalette) {
    if (!svgEl) return;
    const activePalette = customPalette || getActiveThemePalette();
    const clusterColor = activePalette.node2 || '#ffe3a3';
    const clusterBorder = activePalette.ink || '#2b2a33';

    const clusters = svgEl.querySelectorAll ? svgEl.querySelectorAll('.cluster') : [];
    clusters.forEach(cluster => {
      // Find paths in the cluster (e.g. Rough.js hand-drawn shape paths)
      const paths = cluster.querySelectorAll ? cluster.querySelectorAll('path') : [];
      if (paths.length >= 2) {
        // In Rough.js hand-drawn output: Path 0 is the fill shape, Path 1 is the outline border path
        const fillPath = paths[0];
        const strokePath = paths[1];
        if (fillPath) {
          fillPath.style.fill = clusterColor;
          fillPath.style.stroke = 'none';
          if (fillPath.setAttribute) {
            fillPath.setAttribute('fill', clusterColor);
            fillPath.setAttribute('stroke', 'none');
          }
        }
        if (strokePath) {
          strokePath.style.stroke = clusterBorder;
          if (strokePath.setAttribute) {
            strokePath.setAttribute('stroke', clusterBorder);
          }
        }
      } else if (paths.length === 1) {
        paths[0].style.fill = clusterColor;
        if (paths[0].setAttribute) {
          paths[0].setAttribute('fill', clusterColor);
        }
      }

      // Also ensure any rect elements (classic or fallback) are completely filled
      const rects = cluster.querySelectorAll ? cluster.querySelectorAll('rect') : [];
      rects.forEach(r => {
        r.style.fill = clusterColor;
        r.style.stroke = clusterBorder;
        if (r.setAttribute) {
          r.setAttribute('fill', clusterColor);
          r.setAttribute('stroke', clusterBorder);
        }
      });
    });
  }

  function updateTransform() {
    if (activePanZoomController) {
      activePanZoomController.setState(viewState);
      return;
    }
    const vp = document.getElementById('viewport');
    if (vp) {
      vp.setAttribute('transform', `translate(${viewState.x}, ${viewState.y}) scale(${viewState.scale})`);
    }
  }

  function resetView() {
    if (activePanZoomController) {
      activePanZoomController.resetView();
      return;
    }
    viewState.x = 0;
    viewState.y = 0;
    viewState.scale = 1;
    updateTransform();
  }

  function zoom(factor, clientX, clientY) {
    if (activePanZoomController) {
      activePanZoomController.zoom(factor, clientX, clientY);
      return;
    }
    const oldScale = viewState.scale;
    const newScale = Math.min(viewState.maxScale, Math.max(viewState.minScale, oldScale * factor));
    if (newScale === oldScale) return;

    const activeCanvas = document.querySelector('#diagram-stage svg') || document.getElementById('sage-canvas');
    if (clientX !== undefined && clientY !== undefined && activeCanvas) {
      const rect = activeCanvas.getBoundingClientRect();
      const mouseX = clientX - rect.left;
      const mouseY = clientY - rect.top;

      viewState.x = mouseX - ((mouseX - viewState.x) * newScale) / oldScale;
      viewState.y = mouseY - ((mouseY - viewState.y) * newScale) / oldScale;
    }

    viewState.scale = newScale;
    updateTransform();
  }

  function setupActivePanZoom(svgEl, viewportEl) {
    if (activePanZoomController) {
      activePanZoomController.destroy();
      activePanZoomController = null;
    }

    if (typeof PanZoom !== 'undefined' && PanZoom.initPanZoom) {
      activePanZoomController = PanZoom.initPanZoom(svgEl, viewportEl, {
        initialState: viewState,
        onTransform: (newState) => {
          Object.assign(viewState, newState);
        }
      });
    }
  }

  function setupActiveAnnotations(svgEl, viewportEl) {
    if (activeAnnotationsController) {
      activeAnnotationsController.destroy();
      activeAnnotationsController = null;
    }

    if (typeof Annotations !== 'undefined' && Annotations.initAnnotations) {
      activeAnnotationsController = Annotations.initAnnotations(svgEl, viewportEl, {
        initialHighlights: Array.from(persistedHighlights),
        hasMovedDuringDrag: () => {
          return activePanZoomController && activePanZoomController.hasMovedDuringDrag && activePanZoomController.hasMovedDuringDrag();
        },
        onHighlightChange: (activeList) => {
          persistedHighlights.clear();
          activeList.forEach(id => persistedHighlights.add(id));
        }
      });
    }
  }

  function setupPanAndZoom() {
    const stage = document.getElementById('diagram-stage');
    const canvas = (stage && stage.querySelector('svg')) || document.getElementById('sage-canvas');
    const vp = (canvas && canvas.querySelector('#viewport')) || document.getElementById('viewport');
    if (canvas && vp) {
      setupActivePanZoom(canvas, vp);
      setupActiveAnnotations(canvas, vp);
    }

    // Bind static toolbar button listeners
    const btnZoomIn = document.getElementById('btn-zoom-in');
    const btnZoomOut = document.getElementById('btn-zoom-out');
    const btnResetView = document.getElementById('btn-reset-view');
    const btnExportSvg = document.getElementById('btn-export-svg');

    if (btnZoomIn) btnZoomIn.addEventListener('click', () => zoom(1.2));
    if (btnZoomOut) btnZoomOut.addEventListener('click', () => zoom(0.8));
    if (btnResetView) btnResetView.addEventListener('click', resetView);
    if (btnExportSvg) btnExportSvg.addEventListener('click', exportSvg);
  }

  // =========================================================================
  // WARNING BANNER & SVG EXPORT
  // =========================================================================

  function updateWarnings(warnings) {
    if (!warningBanner || !warningMessage) return;
    if (warnings && warnings.length > 0) {
      warningMessage.innerHTML = warnings.map(w => `<div>${escapeHtml(w)}</div>`).join('');
      warningBanner.classList.remove('hidden');
    } else {
      warningBanner.classList.add('hidden');
    }
  }

  function escapeHtml(str) {
    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
  }

  const dismissBtn = typeof document !== 'undefined' ? document.getElementById('btn-dismiss-warning') : null;
  if (dismissBtn && warningBanner) {
    dismissBtn.addEventListener('click', () => warningBanner.classList.add('hidden'));
  }

  function exportSvg() {
    const stage = document.getElementById('diagram-stage');
    const canvas = (stage && stage.querySelector('svg')) || document.getElementById('sage-canvas');
    if (!canvas) return;

    if (typeof SvgExporter !== 'undefined' && SvgExporter.exportDiagramAsSvg) {
      SvgExporter.exportDiagramAsSvg(canvas);
      return;
    }

    const serializer = new XMLSerializer();
    const source = serializer.serializeToString(canvas);
    const blob = new Blob([source], { type: 'image/svg+xml;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'sage-architecture.svg';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  }

  // =========================================================================
  // DOM RENDERING ENGINE (Mermaid Only JS)
  // =========================================================================

  /**
   * Renders Mermaid diagram source code into the target host DOM container.
   * Directly follows the reference implementation in extras/Mermaid × Rough.js comprehensive example.html:
   * uses Mermaid's native handDrawn look without combining with Rough.js post-processing.
   *
   * @param {HTMLElement} host - Target container element for the SVG diagram
   * @param {string} code - Mermaid diagram source code
   * @param {Object} [options] - Diagram rendering parameters (look, seed, colors, font, fontSize)
   */
  async function renderDiagram(host, code, options = {}) {
    if (typeof mermaid === 'undefined') {
      console.error('Mermaid library not loaded');
      return;
    }

    if (!host) {
      host = document.getElementById('diagram-stage');
      if (!host) return;
    }

    lastMermaidSource = code;
    lastDiagramOptions = options;

    const opt = Object.assign({}, DEFAULT_OPTIONS, options);
    const font = opt.font || opt.fontFamily || DEFAULT_OPTIONS.font;
    const fontSize = opt.fontSize || DEFAULT_OPTIONS.fontSize;

    // Await font load prior to mermaid.render so text bounding boxes calculate accurately
    if (typeof document !== 'undefined' && document.fonts && document.fonts.load) {
      try {
        await document.fonts.load(`700 ${fontSize} Caveat`);
        await document.fonts.ready;
      } catch (e) {
        console.warn('Font loading check completed with fallback', e);
      }
    }

    const cfg = mermaidConfig(opt);
    mermaid.initialize(cfg);

    mermaidRenderCounter++;
    const renderId = 'mermaid-svg-' + mermaidRenderCounter;

    try {
      const { svg: svgOutput } = await mermaid.render(renderId, code);
      host.innerHTML = svgOutput;

      const svgEl = host.querySelector('svg');
      if (svgEl) {
        svgEl.id = 'sage-canvas';
        svgEl.classList.add('w-full', 'h-full', 'cursor-grab');

        // Wrap graph content inside dedicated viewport group for pan/zoom
        const viewportEl = wrapInViewport(svgEl);

        // Ensure cluster/subgraph groups have completely solid fills without hachure gaps
        ensureClusterSolidFill(svgEl, cfg.themeVariables ? { node2: cfg.themeVariables.clusterBkg, ink: cfg.themeVariables.clusterBorder } : getActiveThemePalette());

        // Center and scale diagram to fit viewport
        autoFitDiagram(svgEl, viewportEl);

        // Initialize pan and zoom controls
        setupActivePanZoom(svgEl, viewportEl);

        // Initialize click-to-highlight annotations
        setupActiveAnnotations(svgEl, viewportEl);
      }
    } catch (err) {
      console.error('Mermaid rendering failed:', err);
      // Clean up orphaned error element injected by Mermaid on parse failure
      const orphanedErrorEl = document.getElementById('d' + renderId);
      if (orphanedErrorEl) orphanedErrorEl.remove();

      host.innerHTML = '<div class="err p-4 text-xs font-mono text-amber-500 whitespace-pre-wrap"></div>';
      if (host.firstChild) {
        host.firstChild.textContent = 'Could not render: ' + (err.message || err);
      }
      updateWarnings(['Mermaid render error: ' + (err.message || err)]);
    }
  }

  /**
   * Helper alias for renderDiagram that defaults to the #diagram-stage host element.
   */
  async function renderMermaidDiagram(mermaidSource, options = {}) {
    const stage = document.getElementById('diagram-stage');
    return renderDiagram(stage, mermaidSource, options);
  }

  // Common dispatcher for parsed graph AST data and raw Mermaid diagram content
  function applyGraphData(data) {
    if (!data) return;
    updateWarnings(data.warnings);
    if (data.mermaid) {
      renderMermaidDiagram(data.mermaid);
    }
  }

  // =========================================================================
  // LIVE RELOAD (Server-Sent Events)
  // =========================================================================

  function fetchGraphAndRender() {
    fetch('/api/graph')
      .then(resp => {
        if (!resp.ok) throw new Error(`HTTP error ${resp.status}`);
        return resp.json();
      })
      .then(data => {
        applyGraphData(data);
      })
      .catch(err => {
        console.error('Failed to fetch updated graph:', err);
      });
  }

  function setupLiveReload() {
    if (typeof EventSource === 'undefined') return;

    const evtSource = new EventSource('/api/events');

    evtSource.onopen = () => {
      if (liveDot) {
        liveDot.className = 'inline-block w-2 h-2 rounded-full bg-emerald-400 animate-pulse';
      }
      if (liveText) liveText.textContent = 'Live';
    };

    evtSource.onerror = () => {
      if (liveDot) {
        liveDot.className = 'inline-block w-2 h-2 rounded-full bg-amber-400';
      }
      if (liveText) liveText.textContent = 'Reconnecting...';
    };

    evtSource.addEventListener('reload', () => {
      fetchGraphAndRender();
    });
  }

  // Re-render automatically when system color scheme changes (light <-> dark)
  if (typeof window !== 'undefined' && window.matchMedia) {
    const darkModeMatcher = window.matchMedia('(prefers-color-scheme: dark)');
    const handleThemeChange = () => {
      if (lastMermaidSource) {
        renderMermaidDiagram(lastMermaidSource, lastDiagramOptions || {});
      }
    };
    if (darkModeMatcher.addEventListener) {
      darkModeMatcher.addEventListener('change', handleThemeChange);
    } else if (darkModeMatcher.addListener) {
      darkModeMatcher.addListener(handleThemeChange);
    }
  }

  // =========================================================================
  // INITIALIZATION
  // =========================================================================

  function init() {
    setupPanAndZoom();
    setupLiveReload();

    // Load initial data from window variable or JSON script element
    let initialData = (typeof window !== 'undefined' && window.__INITIAL_GRAPH__) ? window.__INITIAL_GRAPH__ : null;
    if (!initialData) {
      const dataScript = document.getElementById('sage-data');
      if (dataScript && dataScript.textContent.trim()) {
        try {
          initialData = JSON.parse(dataScript.textContent);
        } catch (e) {
          console.warn('Could not parse initial sage-data JSON:', e);
        }
      }
    }

    if (initialData && typeof initialData === 'object') {
      applyGraphData(initialData);
    } else {
      fetchGraphAndRender();
    }
  }

  if (typeof document !== 'undefined') {
    if (document.readyState === 'loading') {
      document.addEventListener('DOMContentLoaded', init);
    } else {
      init();
    }
  }

  // =========================================================================
  // EXPORTS
  // =========================================================================

  const exported = {
    DEFAULT_OPTIONS,
    PALETTE,
    getActiveThemePalette,
    getMermaidThemeVariables,
    MERMAID_FONT_FAMILY,
    MERMAID_FONT_SIZE,
    MERMAID_THEME_VARIABLES,
    mermaidConfig,
    renderDiagram,
    renderMermaidDiagram,
    wrapInViewport,
    autoFitDiagram,
    ensureClusterSolidFill,
    viewState
  };

  if (typeof module !== 'undefined' && module.exports) {
    module.exports = exported;
  }
  if (typeof global !== 'undefined') {
    global.SageApp = exported;
  }
})(typeof window !== 'undefined' ? window : this);
