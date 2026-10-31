/**
 * Sage Architecture Visualizer - Frontend Application
 * Ultra-lightweight, zero-build client controller for D2 architecture diagrams.
 * Handles SVG viewport setup, pan/zoom, live-reloading, theme switching, and SVG export.
 */

(function () {
  'use strict';

  // =========================================================================
  // VIEWPORT & STATE
  // ACTIONABLE: Adjust default min/max scale or live reconnect polling.
  // =========================================================================
  const viewState = {
    x: 0,
    y: 0,
    scale: 1,
    minScale: 0.15,
    maxScale: 3.5
  };

  let activePanZoom = null;
  let activeHighlightedEl = null;

  const getEl = id => document.getElementById(id);

  /**
   * Wraps top-level diagram elements in a dedicated <g id="viewport"> for pan/zoom.
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
   * Automatically centers and scales the diagram comfortably within the stage.
   */
  function autoFitDiagram(svgEl, viewportEl) {
    if (!svgEl || !viewportEl) return;
    try {
      const bbox = viewportEl.getBBox();
      if (!bbox || bbox.width <= 0 || bbox.height <= 0) return;

      const stage = document.getElementById('diagram-stage');
      const stageW = (stage && stage.clientWidth) || window.innerWidth || 1280;
      const stageH = (stage && stage.clientHeight) || window.innerHeight || 800;
      const padding = 48;

      const scaleX = (stageW - 2 * padding) / bbox.width;
      const scaleY = (stageH - 2 * padding) / bbox.height;
      let scale = Math.min(scaleX, scaleY, 1.0);
      scale = Math.max(viewState.minScale, Math.min(viewState.maxScale, scale));

      const centerX = bbox.x + bbox.width / 2;
      const centerY = bbox.y + bbox.height / 2;

      const x = (stageW / 2) - (centerX * scale);
      const y = (stageH / 2) - (centerY * scale);

      viewState.x = x;
      viewState.y = y;
      viewState.scale = scale;

      if (activePanZoom) {
        activePanZoom.setState(viewState);
      } else {
        viewportEl.setAttribute('transform', `translate(${x}, ${y}) scale(${scale})`);
      }
    } catch (e) {
      // getBBox might fail in rare headless configurations
    }
  }

  /**
   * Initializes or refreshes pan/zoom and click interaction on the active SVG.
   */
  function setupDiagram(isInitial) {
    const stage = getEl('diagram-stage');
    if (!stage) return;

    const svgEl = stage.querySelector('svg');
    if (!svgEl) return;

    // Remove fixed dimensions from D2 to allow full-stage scaling
    svgEl.style.width = '100%';
    svgEl.style.height = '100%';
    svgEl.style.maxWidth = 'none';

    const viewportEl = wrapInViewport(svgEl);

    if (activePanZoom) {
      activePanZoom.destroy();
      activePanZoom = null;
    }

    if (window.PanZoom && window.PanZoom.initPanZoom) {
      activePanZoom = window.PanZoom.initPanZoom(svgEl, viewportEl, {
        initialState: viewState,
        onTransform: (newState) => {
          Object.assign(viewState, newState);
        },
        btnZoomIn: getEl('btn-zoom-in'),
        btnZoomOut: getEl('btn-zoom-out'),
        btnResetView: getEl('btn-reset-view')
      });
    }

    if (isInitial) {
      autoFitDiagram(svgEl, viewportEl);
    } else if (viewportEl) {
      viewportEl.setAttribute('transform', `translate(${viewState.x}, ${viewState.y}) scale(${viewState.scale})`);
    }

    setupClickHighlight(svgEl);

    // Notify backend that rendering is complete
    fetch('/api/ready').catch(() => {});
  }

  /**
   * Click-to-highlight: highlights selected node and its relations.
   */
  function setupClickHighlight(svgEl) {
    if (!svgEl) return;

    svgEl.addEventListener('click', (e) => {
      if (activePanZoom && activePanZoom.hasMovedDuringDrag && activePanZoom.hasMovedDuringDrag()) {
        return;
      }

      // Find closest shape container or interactive element
      const target = e.target.closest('g') || e.target;
      if (target === svgEl || target.id === 'viewport') {
        clearHighlight(svgEl);
        return;
      }

      if (activeHighlightedEl === target) {
        clearHighlight(svgEl);
        return;
      }

      highlightElement(svgEl, target);
    });
  }

  function clearHighlight(svgEl) {
    activeHighlightedEl = null;
    const allEls = svgEl.querySelectorAll('.d2-dimmed, .d2-highlighted');
    allEls.forEach(el => {
      el.classList.remove('d2-dimmed', 'd2-highlighted');
    });
  }

  function highlightElement(svgEl, target) {
    activeHighlightedEl = target;
    const viewport = svgEl.querySelector('#viewport') || svgEl;
    const directChildren = Array.from(viewport.children);

    directChildren.forEach(child => {
      if (child.contains(target) || child === target) {
        child.classList.remove('d2-dimmed');
        child.classList.add('d2-highlighted');
      } else {
        child.classList.remove('d2-highlighted');
        child.classList.add('d2-dimmed');
      }
    });
  }

  /**
   * Updates diagnostic warnings banner.
   */
  function updateWarnings(warnings) {
    const banner = getEl('warning-banner');
    const msg = getEl('warning-message');
    if (!banner || !msg) return;

    if (warnings && warnings.length > 0) {
      msg.innerHTML = warnings.map(w => `<div>${escapeHtml(w)}</div>`).join('');
      banner.classList.remove('hidden');
    } else {
      banner.classList.add('hidden');
    }
  }

  function escapeHtml(str) {
    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
  }

  /**
   * Export diagram as SVG file download.
   */
  function exportSvg() {
    const stage = getEl('diagram-stage');
    const svgEl = stage ? stage.querySelector('svg') : null;
    if (!svgEl) return;

    const clone = svgEl.cloneNode(true);
    const serializer = new XMLSerializer();
    const source = serializer.serializeToString(clone);
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

  /**
   * Live-reload via Server-Sent Events (/api/events).
   */
  function setupLiveReload() {
    if (typeof EventSource === 'undefined') return;

    const liveDot = getEl('live-dot');
    const liveText = getEl('live-text');
    const evtSource = new EventSource('/api/events');

    evtSource.onopen = () => {
      if (liveDot) liveDot.className = 'inline-block w-2 h-2 rounded-full bg-emerald-400 animate-pulse';
      if (liveText) liveText.textContent = 'Live';
    };

    evtSource.onerror = () => {
      if (liveDot) liveDot.className = 'inline-block w-2 h-2 rounded-full bg-amber-400';
      if (liveText) liveText.textContent = 'Reconnecting...';
    };

    evtSource.addEventListener('reload', () => {
      fetchDiagram();
    });
  }

  function fetchDiagram() {
    fetch('/api/diagram')
      .then(res => res.json())
      .then(data => {
        if (!data) return;
        updateWarnings(data.warnings);
        if (data.svg) {
          const stage = getEl('diagram-stage');
          if (stage) {
            stage.innerHTML = data.svg;
            setupDiagram(false);
          }
        }
      })
      .catch(err => {
        console.error('Failed to reload diagram:', err);
      });
  }

  /**
   * Theme switcher: dark / light mode.
   */
  function setupTheme() {
    const btnTheme = getEl('btn-theme-toggle');
    if (!btnTheme) return;

    try {
      const saved = localStorage.getItem('sage-theme');
      if (saved) {
        document.documentElement.setAttribute('data-theme', saved);
      }
    } catch (e) {}

    btnTheme.addEventListener('click', () => {
      const current = document.documentElement.getAttribute('data-theme');
      const isDark = current === 'dark' || (!current && window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches);
      const nextTheme = isDark ? 'light' : 'dark';
      document.documentElement.setAttribute('data-theme', nextTheme);
      try { localStorage.setItem('sage-theme', nextTheme); } catch (e) {}
    });
  }

  // =========================================================================
  // BOOTSTRAP
  // =========================================================================
  function init() {
    setupTheme();
    setupDiagram(true);
    setupLiveReload();

    const btnExport = getEl('btn-export-svg');
    if (btnExport) {
      btnExport.addEventListener('click', exportSvg);
    }
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
