/**
 * Sage Architecture Visualizer - Rough Annotations Module
 *
 * Provides click-to-highlight functionality for architecture nodes, encircling
 * selected components with a hand-drawn Rough.js sketchy ring that transforms
 * in lockstep with the viewport during pan and zoom navigation.
 */

// =============================================================================
// VISUAL TUNING VARIABLES FOR ANNOTATIONS
// TWEAK HERE: Adjust ring accent color, roughness, thickness, and padding:
// =============================================================================
const DEFAULT_ANNOTATION_CONFIG = {
  // Accent stroke color for the sketchy ring (coral/orange standout for dark slate theme)
  stroke: '#ff7a59',

  // Thickness of the hand-drawn sketchy ring line
  strokeWidth: 2.4,

  // Roughness: 1.8 gives an energetic, organic hand-drawn circular stroke
  roughness: 1.8,

  // Horizontal and vertical padding added around node bounding box
  paddingX: 24,
  paddingY: 20,

  // Base random seed for reproducible sketchy shapes
  seed: 50
};

/**
 * Creates a new annotation state tracker.
 */
function createAnnotationState(initialItems = []) {
  return {
    activeIds: new Set(initialItems)
  };
}

/**
 * Toggles highlight state for a given node ID.
 * Returns { active: boolean, id: string }.
 */
function toggleHighlight(state, nodeId) {
  if (!state || !nodeId) return { active: false, id: nodeId };

  if (state.activeIds.has(nodeId)) {
    state.activeIds.delete(nodeId);
    return { active: false, id: nodeId };
  } else {
    state.activeIds.add(nodeId);
    return { active: true, id: nodeId };
  }
}

/**
 * Checks if a node is currently highlighted.
 */
function isHighlighted(state, nodeId) {
  if (!state || !nodeId) return false;
  return state.activeIds.has(nodeId);
}

/**
 * Clears all active annotations.
 */
function clearHighlights(state) {
  if (state && state.activeIds) {
    state.activeIds.clear();
  }
}

/**
 * Extracts a clean, canonical node ID from Mermaid node DOM elements.
 * Example: 'flowchart-apiGateway-12' -> 'apiGateway'
 * Example: 'state-idle-4' -> 'idle'
 */
function extractNodeId(el) {
  if (!el) return '';
  const rawId = (el.getAttribute && (el.getAttribute('data-id') || el.getAttribute('id'))) || el.id || '';
  const match = /^(?:flowchart|state)-(.+)-\d+$/.exec(rawId);
  if (match) return match[1];
  return rawId;
}

/**
 * Computes bounding box of an element transformed into viewport coordinate space.
 *
 * @param {Object} clientRect - Screen rect { left, top, right, bottom } from getBoundingClientRect().
 * @param {Object} inverseMatrix - Inverse CTM matrix or transform helper.
 */
function computeNodeBounds(clientRect, inverseMatrix) {
  let p1 = { x: clientRect.left, y: clientRect.top };
  let p2 = { x: clientRect.right, y: clientRect.bottom };

  if (inverseMatrix) {
    if (typeof inverseMatrix.transformPoint === 'function') {
      p1 = inverseMatrix.transformPoint(p1);
      p2 = inverseMatrix.transformPoint(p2);
    } else if (typeof DOMPoint !== 'undefined' && inverseMatrix.a !== undefined) {
      p1 = new DOMPoint(clientRect.left, clientRect.top).matrixTransform(inverseMatrix);
      p2 = new DOMPoint(clientRect.right, clientRect.bottom).matrixTransform(inverseMatrix);
    }
  }

  const x = Math.min(p1.x, p2.x);
  const y = Math.min(p1.y, p2.y);
  const w = Math.abs(p2.x - p1.x);
  const h = Math.abs(p2.y - p1.y);

  return {
    x,
    y,
    w,
    h,
    cx: x + w / 2,
    cy: y + h / 2
  };
}

/**
 * Computes Rough.js options and geometry dimensions for the sketchy highlight ring.
 */
function createRoughRingOptions(box, config = {}, seedOffset = 0) {
  const cfg = Object.assign({}, DEFAULT_ANNOTATION_CONFIG, config);
  const ringWidth = box.w + (cfg.paddingX || 24);
  const ringHeight = box.h + (cfg.paddingY || 20);

  return {
    stroke: cfg.stroke,
    strokeWidth: cfg.strokeWidth,
    roughness: cfg.roughness,
    seed: (cfg.seed || 50) + seedOffset,
    ringWidth,
    ringHeight
  };
}

/**
 * Initializes interactive click-to-highlight annotations on an SVG stage and viewport.
 *
 * @param {SVGSVGElement} svg - The SVG element containing the diagram.
 * @param {SVGElement} viewportGroup - The group that pan/zoom transforms.
 * @param {Object} [options] - Optional custom configuration and callbacks.
 */
function initAnnotations(svg, viewportGroup, options = {}) {
  if (!svg || !viewportGroup) return null;

  const config = Object.assign({}, DEFAULT_ANNOTATION_CONFIG, options.config || {});
  const state = createAnnotationState(options.initialHighlights || []);
  const roughLib = typeof rough !== 'undefined' ? rough : (typeof window !== 'undefined' ? window.rough : null);

  // Ensure dedicated annotations layer exists inside the viewport group
  let annotLayer = viewportGroup.querySelector('#annotations-layer');
  if (!annotLayer) {
    annotLayer = document.createElementNS('http://www.w3.org/2000/svg', 'g');
    annotLayer.setAttribute('id', 'annotations-layer');
    annotLayer.setAttribute('class', 'annot');
    annotLayer.style.pointerEvents = 'none';
    viewportGroup.appendChild(annotLayer);
  }

  function getNodesMap() {
    const map = new Map();
    const nodeElements = svg.querySelectorAll('g.node, g[class*="node"]');
    nodeElements.forEach(el => {
      const id = extractNodeId(el);
      if (id) {
        map.set(id, el);
      }
    });
    return map;
  }

  function render() {
    if (!annotLayer) return;
    annotLayer.innerHTML = '';
    if (!roughLib) return;

    const rc = roughLib.svg(svg);
    const nodes = getNodesMap();
    let seedOffset = 0;

    state.activeIds.forEach(nodeId => {
      const nodeEl = nodes.get(nodeId);
      if (!nodeEl) return;

      const rect = nodeEl.getBoundingClientRect();
      const inverseCTM = viewportGroup.getScreenCTM ? viewportGroup.getScreenCTM().inverse() : null;
      const box = computeNodeBounds(rect, inverseCTM);
      const ringOpts = createRoughRingOptions(box, config, seedOffset++);

      const sketchyRing = rc.ellipse(box.cx, box.cy, ringOpts.ringWidth, ringOpts.ringHeight, {
        stroke: ringOpts.stroke,
        strokeWidth: ringOpts.strokeWidth,
        roughness: ringOpts.roughness,
        seed: ringOpts.seed,
        disableMultiStroke: false
      });

      sketchyRing.setAttribute('data-annotation-for', nodeId);
      sketchyRing.setAttribute('class', 'rough-ring-highlight');
      annotLayer.appendChild(sketchyRing);
    });

    if (typeof options.onHighlightChange === 'function') {
      options.onHighlightChange(Array.from(state.activeIds));
    }
  }

  function onClick(e) {
    if (options.hasMovedDuringDrag && options.hasMovedDuringDrag()) {
      return;
    }

    const nodeEl = e.target.closest('g.node, g[class*="node"]');
    if (!nodeEl) return;

    const nodeId = extractNodeId(nodeEl);
    if (!nodeId) return;

    toggleHighlight(state, nodeId);
    render();
  }

  svg.addEventListener('click', onClick);

  function updateNodeCursors() {
    const nodes = svg.querySelectorAll('g.node, g[class*="node"]');
    nodes.forEach(n => {
      n.style.cursor = 'pointer';
    });
  }

  updateNodeCursors();
  render();

  return {
    getState: () => state,
    toggle: (nodeId) => {
      const res = toggleHighlight(state, nodeId);
      render();
      return res;
    },
    clear: () => {
      clearHighlights(state);
      render();
    },
    render,
    updateNodeCursors,
    destroy: () => {
      svg.removeEventListener('click', onClick);
      if (annotLayer && annotLayer.parentNode) {
        annotLayer.parentNode.removeChild(annotLayer);
      }
    }
  };
}

if (typeof module !== 'undefined' && module.exports) {
  module.exports = {
    DEFAULT_ANNOTATION_CONFIG,
    createAnnotationState,
    toggleHighlight,
    isHighlighted,
    clearHighlights,
    extractNodeId,
    computeNodeBounds,
    createRoughRingOptions,
    initAnnotations
  };
}

if (typeof window !== 'undefined') {
  window.Annotations = {
    DEFAULT_ANNOTATION_CONFIG,
    createAnnotationState,
    toggleHighlight,
    isHighlighted,
    clearHighlights,
    extractNodeId,
    computeNodeBounds,
    createRoughRingOptions,
    initAnnotations
  };
}
