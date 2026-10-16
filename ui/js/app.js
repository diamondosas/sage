/**
 * Sage Architecture Visualizer Frontend Application
 *
 * Renders an Excalidraw-style architectural diagram using Dagre auto-layout
 * and Rough.js in SVG mode with live reload, pan, zoom, and draggable nodes.
 */

(function () {
  'use strict';

  // =========================================================================
  // VISUAL VARIABLES & CONFIGURATION
  // Tweak these variables to customize the appearance and behavior of the graph.
  // =========================================================================

  // Layout spacing configurations for Dagre auto-layout.
  // Increase 'nodesep' for wider spacing between parallel nodes.
  // Increase 'ranksep' for greater distance between dependency tiers.
  const LAYOUT_CONFIG = {
    rankdir: 'LR',   // 'LR' = left-to-right, 'TB' = top-to-bottom
    align: 'UL',
    nodesep: 60,     // Horizontal node separation in pixels
    ranksep: 80,     // Vertical/rank separation in pixels
    marginx: 50,
    marginy: 50
  };

  // Rough.js sketchy drawing parameters.
  // - roughness: 0.0 is completely clean/flat; 1.5 is standard sketchy; 2.5 is very rough.
  // - bowing: Curvature deviation of straight lines.
  // - fillStyle: 'hachure', 'solid', 'cross-hatch', 'dots', or 'zigzag'.
  const ROUGH_DEFAULTS = {
    roughness: 1.4,
    bowing: 1.0,
    strokeWidth: 1.6,
    fillStyle: 'hachure',
    fillWeight: 1.2,
    hachureAngle: 60,
    hachureGap: 6
  };

  // Color palette definitions for node outlines, fills, and labels.
  // You can modify or add new palette keys matching the [color] DSL tags.
  const COLOR_PALETTE = {
    blue:   { stroke: '#60a5fa', fill: '#1e3a8a', text: '#dbeafe' },
    green:  { stroke: '#4ade80', fill: '#14532d', text: '#dcfce7' },
    red:    { stroke: '#f87171', fill: '#7f1d1d', text: '#fee2e2' },
    yellow: { stroke: '#facc15', fill: '#713f12', text: '#fef9c3' },
    purple: { stroke: '#c084fc', fill: '#581c87', text: '#f3e8ff' },
    orange: { stroke: '#fb923c', fill: '#7c2d12', text: '#ffedd5' },
    gray:   { stroke: '#94a3b8', fill: '#334155', text: '#f1f5f9' },
    default:{ stroke: '#cbd5e1', fill: '#1e293b', text: '#f8fafc' }
  };

  // Node dimensions based on shape type
  const SHAPE_DIMENSIONS = {
    rect:     { width: 160, height: 60 },
    circle:   { diameter: 80 },
    diamond:  { width: 140, height: 80 },
    triangle: { width: 150, height: 75 },
    cylinder: { width: 150, height: 70 }
  };

  // =========================================================================
  // STATE MANAGEMENT
  // =========================================================================

  let currentGraph = null;
  let layoutNodes = new Map(); // id -> { x, y, width, height, shape, label, color, element }
  let layoutEdges = [];        // array of { from, to, label, bidirectional, element, pathElement }

  // Pan and zoom transformation state
  const viewState = {
    x: 0,
    y: 0,
    scale: 1,
    minScale: 0.15,
    maxScale: 3.0
  };

  // DOM Elements
  const svg = document.getElementById('sage-canvas');
  const viewport = document.getElementById('viewport');
  const edgesLayer = document.getElementById('edges-layer');
  const nodesLayer = document.getElementById('nodes-layer');
  const warningBanner = document.getElementById('warning-banner');
  const warningMessage = document.getElementById('warning-message');
  const liveDot = document.getElementById('live-dot');
  const liveText = document.getElementById('live-text');

  // Rough.js SVG instance
  let rc = null;
  if (typeof rough !== 'undefined' && svg) {
    rc = rough.svg(svg);
  }

  // =========================================================================
  // GRAPH AUTO-LAYOUT (Dagre)
  // =========================================================================

  /**
   * Computes 2D node coordinates and edge routes using Dagre layout engine.
   */
  function computeLayout(graphData) {
    if (typeof dagre === 'undefined') {
      console.error('Dagre library is not loaded');
      return;
    }

    const g = new dagre.graphlib.Graph();
    g.setGraph(LAYOUT_CONFIG);
    g.setDefaultEdgeLabel(() => ({}));

    // Add nodes to Dagre graph
    (graphData.nodes || []).forEach(node => {
      const shape = node.shape || 'rect';
      let w = 160;
      let h = 60;

      if (shape === 'circle') {
        w = SHAPE_DIMENSIONS.circle.diameter;
        h = SHAPE_DIMENSIONS.circle.diameter;
      } else if (SHAPE_DIMENSIONS[shape]) {
        w = SHAPE_DIMENSIONS[shape].width;
        h = SHAPE_DIMENSIONS[shape].height;
      }

      // Dynamically expand width if label is long
      const labelText = node.label || node.id;
      if (labelText.length * 8.5 > w) {
        w = Math.round(labelText.length * 8.5) + 30;
      }

      g.setNode(node.id, {
        label: labelText,
        width: w,
        height: h,
        shape: shape,
        color: node.color || 'default',
        group: node.group || '',
        filePath: node.filePath || ''
      });
    });

    // Add edges to Dagre graph
    (graphData.edges || []).forEach(edge => {
      g.setEdge(edge.from, edge.to, {
        label: edge.label || '',
        bidirectional: !!edge.bidirectional
      });
    });

    // Run layout calculation
    dagre.layout(g);

    // Save computed layout nodes into state map
    layoutNodes.clear();
    g.nodes().forEach(nodeId => {
      const node = g.node(nodeId);
      if (node) {
        layoutNodes.set(nodeId, {
          id: nodeId,
          x: node.x,
          y: node.y,
          width: node.width,
          height: node.height,
          shape: node.shape,
          label: node.label,
          color: node.color,
          group: node.group,
          filePath: node.filePath
        });
      }
    });

    // Save computed edges
    layoutEdges = [];
    g.edges().forEach(e => {
      const edge = g.edge(e);
      layoutEdges.push({
        from: e.v,
        to: e.w,
        label: edge.label || '',
        bidirectional: !!edge.bidirectional,
        points: edge.points || []
      });
    });
  }

  // =========================================================================
  // ROUGH.JS SVG RENDERING
  // =========================================================================

  /**
   * Renders a hand-drawn node shape using Rough.js in SVG mode.
   */
  function renderNodeShape(node) {
    const palette = COLOR_PALETTE[node.color] || COLOR_PALETTE.default;
    const options = {
      ...ROUGH_DEFAULTS,
      stroke: palette.stroke,
      fill: palette.fill
    };

    const halfW = node.width / 2;
    const halfH = node.height / 2;
    const x = -halfW;
    const y = -halfH;

    const group = document.createElementNS('http://www.w3.org/2000/svg', 'g');
    group.setAttribute('class', 'node-shape');

    let shapeElement = null;

    switch (node.shape) {
      case 'circle': {
        const radius = Math.min(node.width, node.height) / 2;
        shapeElement = rc.circle(0, 0, radius * 2, options);
        break;
      }
      case 'diamond': {
        const points = [
          [0, y],
          [halfW, 0],
          [0, halfH],
          [-halfW, 0]
        ];
        shapeElement = rc.polygon(points, options);
        break;
      }
      case 'triangle': {
        const points = [
          [0, y],
          [halfW, halfH],
          [-halfW, halfH]
        ];
        shapeElement = rc.polygon(points, options);
        break;
      }
      case 'cylinder': {
        const ry = 10;
        const topEllipse = rc.ellipse(0, y + ry, node.width, ry * 2, options);
        const bottomArc = rc.arc(0, halfH - ry, node.width, ry * 2, 0, Math.PI, false, options);
        const leftLine = rc.line(-halfW, y + ry, -halfW, halfH - ry, options);
        const rightLine = rc.line(halfW, y + ry, halfW, halfH - ry, options);

        group.appendChild(topEllipse);
        group.appendChild(bottomArc);
        group.appendChild(leftLine);
        group.appendChild(rightLine);
        break;
      }
      case 'rect':
      default: {
        shapeElement = rc.rectangle(x, y, node.width, node.height, options);
        break;
      }
    }

    if (shapeElement) {
      group.appendChild(shapeElement);
    }

    // Centered label text
    const text = document.createElementNS('http://www.w3.org/2000/svg', 'text');
    text.setAttribute('x', '0');
    // For triangles, slightly bias text down for better visual centering
    text.setAttribute('y', node.shape === 'triangle' ? '8' : '0');
    text.setAttribute('text-anchor', 'middle');
    text.setAttribute('dominant-baseline', 'central');
    text.setAttribute('fill', palette.text);
    text.setAttribute('font-size', '13px');
    text.setAttribute('font-weight', '500');
    text.setAttribute('pointer-events', 'none');
    text.textContent = node.label || node.id;
    group.appendChild(text);

    return group;
  }

  /**
   * Generates a curved or straight SVG path string for an edge connecting points.
   */
  function buildPathString(points) {
    if (!points || points.length === 0) return '';
    if (points.length === 2) {
      return `M ${points[0].x} ${points[0].y} L ${points[1].x} ${points[1].y}`;
    }

    // Build smooth quadratic/cubic Bezier curve through points
    let d = `M ${points[0].x} ${points[0].y}`;
    for (let i = 1; i < points.length - 1; i++) {
      const xc = (points[i].x + points[i + 1].x) / 2;
      const yc = (points[i].y + points[i + 1].y) / 2;
      d += ` Q ${points[i].x} ${points[i].y}, ${xc} ${yc}`;
    }
    const last = points[points.length - 1];
    d += ` T ${last.x} ${last.y}`;
    return d;
  }

  /**
   * Renders an edge connector between two nodes.
   */
  function renderEdge(edge) {
    const fromNode = layoutNodes.get(edge.from);
    const toNode = layoutNodes.get(edge.to);
    if (!fromNode || !toNode) return null;

    const group = document.createElementNS('http://www.w3.org/2000/svg', 'g');
    group.setAttribute('class', 'edge');
    group.setAttribute('data-from', edge.from);
    group.setAttribute('data-to', edge.to);

    // If points exist from Dagre, use them; otherwise direct line between centers
    let points = edge.points;
    if (!points || points.length < 2) {
      points = [{ x: fromNode.x, y: fromNode.y }, { x: toNode.x, y: toNode.y }];
    }

    const pathString = buildPathString(points);

    // Draw rough sketchy path
    const roughPath = rc.path(pathString, {
      ...ROUGH_DEFAULTS,
      stroke: '#94a3b8',
      strokeWidth: 1.5,
      roughness: 1.0
    });

    // Underlying SVG path for clean arrowhead marker attachment
    const guidePath = document.createElementNS('http://www.w3.org/2000/svg', 'path');
    guidePath.setAttribute('d', pathString);
    guidePath.setAttribute('fill', 'none');
    guidePath.setAttribute('stroke', 'transparent');
    guidePath.setAttribute('stroke-width', '2');
    guidePath.setAttribute('marker-end', 'url(#arrowhead)');
    if (edge.bidirectional) {
      guidePath.setAttribute('marker-start', 'url(#arrowhead-start)');
    }

    group.appendChild(roughPath);
    group.appendChild(guidePath);

    // Optional edge label
    if (edge.label) {
      const midIdx = Math.floor(points.length / 2);
      const midPoint = points[midIdx] || { x: (fromNode.x + toNode.x) / 2, y: (fromNode.y + toNode.y) / 2 };

      const labelGroup = document.createElementNS('http://www.w3.org/2000/svg', 'g');
      labelGroup.setAttribute('class', 'edge-label');
      labelGroup.setAttribute('transform', `translate(${midPoint.x}, ${midPoint.y - 10})`);

      const labelBg = document.createElementNS('http://www.w3.org/2000/svg', 'rect');
      const textWidth = Math.max(40, edge.label.length * 7 + 12);
      labelBg.setAttribute('x', -textWidth / 2);
      labelBg.setAttribute('y', -10);
      labelBg.setAttribute('width', textWidth);
      labelBg.setAttribute('height', 20);
      labelBg.setAttribute('rx', 4);
      labelBg.setAttribute('fill', '#0f172a');
      labelBg.setAttribute('stroke', '#475569');
      labelBg.setAttribute('stroke-width', '1');

      const labelText = document.createElementNS('http://www.w3.org/2000/svg', 'text');
      labelText.setAttribute('text-anchor', 'middle');
      labelText.setAttribute('dominant-baseline', 'central');
      labelText.setAttribute('fill', '#cbd5e1');
      labelText.setAttribute('font-size', '11px');
      labelText.textContent = edge.label;

      labelGroup.appendChild(labelBg);
      labelGroup.appendChild(labelText);
      group.appendChild(labelGroup);
    }

    return group;
  }

  /**
   * Re-renders all edges attached to a specific node (used during dragging).
   */
  function updateConnectedEdges(nodeId) {
    layoutEdges.forEach(edge => {
      if (edge.from === nodeId || edge.to === nodeId) {
        const fromNode = layoutNodes.get(edge.from);
        const toNode = layoutNodes.get(edge.to);
        if (!fromNode || !toNode) return;

        // Recalculate 2-point direct route while dragging
        edge.points = [{ x: fromNode.x, y: fromNode.y }, { x: toNode.x, y: toNode.y }];

        if (edge.element && edge.element.parentNode) {
          const newEdgeEl = renderEdge(edge);
          if (newEdgeEl) {
            edge.element.parentNode.replaceChild(newEdgeEl, edge.element);
            edge.element = newEdgeEl;
          }
        }
      }
    });
  }

  /**
   * Clears and re-renders the complete graph into the SVG viewport.
   */
  function renderGraph() {
    if (!edgesLayer || !nodesLayer || !rc) return;

    edgesLayer.innerHTML = '';
    nodesLayer.innerHTML = '';

    // Render edges first so they sit below nodes
    layoutEdges.forEach(edge => {
      const edgeEl = renderEdge(edge);
      if (edgeEl) {
        edge.element = edgeEl;
        edgesLayer.appendChild(edgeEl);
      }
    });

    // Render nodes
    layoutNodes.forEach(node => {
      const nodeGroup = document.createElementNS('http://www.w3.org/2000/svg', 'g');
      nodeGroup.setAttribute('class', 'node cursor-grab active:cursor-grabbing');
      nodeGroup.setAttribute('data-id', node.id);
      nodeGroup.setAttribute('transform', `translate(${node.x}, ${node.y})`);

      const shapeEl = renderNodeShape(node);
      nodeGroup.appendChild(shapeEl);

      node.element = nodeGroup;
      nodesLayer.appendChild(nodeGroup);

      // Attach node drag listeners
      setupNodeDragging(nodeGroup, node);
    });
  }

  // =========================================================================
  // INTERACTION: PAN & ZOOM
  // =========================================================================

  function updateTransform() {
    if (!viewport) return;
    viewport.setAttribute('transform', `translate(${viewState.x}, ${viewState.y}) scale(${viewState.scale})`);
  }

  function resetView() {
    viewState.x = 0;
    viewState.y = 0;
    viewState.scale = 1;
    updateTransform();
  }

  function zoom(factor, clientX, clientY) {
    const oldScale = viewState.scale;
    const newScale = Math.min(viewState.maxScale, Math.max(viewState.minScale, oldScale * factor));
    if (newScale === oldScale) return;

    if (clientX !== undefined && clientY !== undefined && svg) {
      const rect = svg.getBoundingClientRect();
      const mouseX = clientX - rect.left;
      const mouseY = clientY - rect.top;

      viewState.x = mouseX - ((mouseX - viewState.x) * newScale) / oldScale;
      viewState.y = mouseY - ((mouseY - viewState.y) * newScale) / oldScale;
    }

    viewState.scale = newScale;
    updateTransform();
  }

  function setupPanAndZoom() {
    if (!svg) return;

    let isPanning = false;
    let startX = 0;
    let startY = 0;

    svg.addEventListener('mousedown', e => {
      // Only pan if clicking canvas background (not on a node)
      if (e.target.closest('.node')) return;
      isPanning = true;
      startX = e.clientX - viewState.x;
      startY = e.clientY - viewState.y;
    });

    window.addEventListener('mousemove', e => {
      if (!isPanning) return;
      viewState.x = e.clientX - startX;
      viewState.y = e.clientY - startY;
      updateTransform();
    });

    window.addEventListener('mouseup', () => {
      isPanning = false;
    });

    // Mouse wheel zoom
    svg.addEventListener('wheel', e => {
      e.preventDefault();
      const factor = e.deltaY < 0 ? 1.1 : 0.9;
      zoom(factor, e.clientX, e.clientY);
    }, { passive: false });

    // Header buttons
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
  // INTERACTION: NODE DRAGGING
  // =========================================================================

  function setupNodeDragging(nodeElement, node) {
    let isDragging = false;
    let dragStartX = 0;
    let dragStartY = 0;
    let initialNodeX = 0;
    let initialNodeY = 0;

    nodeElement.addEventListener('mousedown', e => {
      e.stopPropagation();
      isDragging = true;
      dragStartX = e.clientX;
      dragStartY = e.clientY;
      initialNodeX = node.x;
      initialNodeY = node.y;
    });

    window.addEventListener('mousemove', e => {
      if (!isDragging) return;
      const dx = (e.clientX - dragStartX) / viewState.scale;
      const dy = (e.clientY - dragStartY) / viewState.scale;

      node.x = initialNodeX + dx;
      node.y = initialNodeY + dy;

      nodeElement.setAttribute('transform', `translate(${node.x}, ${node.y})`);
      updateConnectedEdges(node.id);
    });

    window.addEventListener('mouseup', () => {
      if (isDragging) {
        isDragging = false;
      }
    });
  }

  // =========================================================================
  // WARNING BANNER & EXPORT
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

  const dismissBtn = document.getElementById('btn-dismiss-warning');
  if (dismissBtn && warningBanner) {
    dismissBtn.addEventListener('click', () => warningBanner.classList.add('hidden'));
  }

  function exportSvg() {
    if (!svg) return;
    const serializer = new XMLSerializer();
    const source = serializer.serializeToString(svg);
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
  // LIVE RELOAD (Server-Sent Events)
  // =========================================================================

  function fetchGraphAndRender() {
    fetch('/api/graph')
      .then(resp => {
        if (!resp.ok) throw new Error(`HTTP error ${resp.status}`);
        return resp.json();
      })
      .then(data => {
        currentGraph = data;
        updateWarnings(data.warnings);
        computeLayout(data);
        renderGraph();
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

  // =========================================================================
  // INITIALIZATION
  // =========================================================================

  function init() {
    setupPanAndZoom();
    setupLiveReload();

    // Load initial data from window variable or JSON script element
    let initialData = window.__INITIAL_GRAPH__;
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
      currentGraph = initialData;
      updateWarnings(initialData.warnings);
      computeLayout(initialData);
      renderGraph();
    } else {
      // Fallback: fetch from /api/graph if initial data wasn't inlined
      fetchGraphAndRender();
    }
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }

})();
