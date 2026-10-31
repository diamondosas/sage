/**
 * Sage Architecture Visualizer - Pan and Zoom Controller
 * Provides smooth canvas panning by dragging, mouse wheel zooming with bounded
 * scaling limits, reset coordinates, and toolbar button bindings.
 */

// =============================================================================
// PAN & ZOOM CONFIGURATION
// ACTIONABLE: Adjust minScale, maxScale, or zoom step factors here.
// =============================================================================
const DEFAULT_VIEW_CONFIG = {
  x: 0,
  y: 0,
  scale: 1,
  minScale: 0.15,
  maxScale: 3.5,
  zoomInFactor: 1.2,
  zoomOutFactor: 0.8
};

function createViewState(overrides = {}) {
  return Object.assign({}, DEFAULT_VIEW_CONFIG, overrides);
}

function clampScale(scale, minScale = 0.15, maxScale = 3.5) {
  return Math.min(maxScale, Math.max(minScale, scale));
}

function calculateZoom(currentViewState, factor, focalPoint, stageBounds) {
  const oldScale = currentViewState.scale;
  const newScale = clampScale(oldScale * factor, currentViewState.minScale, currentViewState.maxScale);

  if (newScale === oldScale) {
    return Object.assign({}, currentViewState);
  }

  let mouseX = 0;
  let mouseY = 0;

  if (focalPoint && stageBounds) {
    mouseX = focalPoint.clientX - stageBounds.left;
    mouseY = focalPoint.clientY - stageBounds.top;
  } else if (stageBounds) {
    mouseX = stageBounds.width / 2;
    mouseY = stageBounds.height / 2;
  }

  const newX = mouseX - ((mouseX - currentViewState.x) * newScale) / oldScale;
  const newY = mouseY - ((mouseY - currentViewState.y) * newScale) / oldScale;

  return Object.assign({}, currentViewState, {
    x: newX,
    y: newY,
    scale: newScale
  });
}

function calculatePan(currentViewState, dx, dy) {
  return Object.assign({}, currentViewState, {
    x: currentViewState.x + dx,
    y: currentViewState.y + dy
  });
}

function calculateReset(currentViewState) {
  return Object.assign({}, currentViewState, {
    x: 0,
    y: 0,
    scale: 1
  });
}

function getTransformString(viewState) {
  return `translate(${viewState.x}, ${viewState.y}) scale(${viewState.scale})`;
}

function applyTransform(targetElement, viewState) {
  if (!targetElement) return;
  targetElement.setAttribute('transform', getTransformString(viewState));
}

function initPanZoom(svg, viewport, options = {}) {
  if (!svg || !viewport) return null;

  let state = createViewState(options.initialState || {});
  let isDragging = false;
  let dragStartX = 0;
  let dragStartY = 0;
  let startX = 0;
  let startY = 0;
  let hasMoved = false;

  const onTransform = options.onTransform || (() => {});

  function update() {
    applyTransform(viewport, state);
    onTransform(state);
  }

  update();

  function onMouseDown(e) {
    if (e.button !== 0) return;

    isDragging = true;
    hasMoved = false;
    dragStartX = e.clientX;
    dragStartY = e.clientY;
    startX = e.clientX - state.x;
    startY = e.clientY - state.y;

    svg.style.cursor = 'grabbing';
  }

  function onMouseMove(e) {
    if (!isDragging) return;

    const moveDistance = Math.hypot(e.clientX - dragStartX, e.clientY - dragStartY);
    if (moveDistance > 3) {
      hasMoved = true;
    }

    state.x = e.clientX - startX;
    state.y = e.clientY - startY;
    update();
  }

  function onMouseUp() {
    if (isDragging) {
      isDragging = false;
      svg.style.cursor = 'grab';
    }
  }

  function onWheel(e) {
    e.preventDefault();
    const factor = e.deltaY < 0 ? (options.zoomInFactor || 1.1) : (options.zoomOutFactor || 0.9);
    const rect = svg.getBoundingClientRect();
    state = calculateZoom(state, factor, { clientX: e.clientX, clientY: e.clientY }, rect);
    update();
  }

  svg.addEventListener('mousedown', onMouseDown);
  window.addEventListener('mousemove', onMouseMove);
  window.addEventListener('mouseup', onMouseUp);
  svg.addEventListener('wheel', onWheel, { passive: false });

  const btnZoomIn = options.btnZoomIn || null;
  const btnZoomOut = options.btnZoomOut || null;
  const btnResetView = options.btnResetView || null;

  const zoomInHandler = () => {
    const rect = svg.getBoundingClientRect();
    state = calculateZoom(state, options.zoomInFactor || 1.2, null, rect);
    update();
  };

  const zoomOutHandler = () => {
    const rect = svg.getBoundingClientRect();
    state = calculateZoom(state, options.zoomOutFactor || 0.8, null, rect);
    update();
  };

  const resetHandler = () => {
    state = calculateReset(state);
    update();
  };

  if (btnZoomIn) btnZoomIn.addEventListener('click', zoomInHandler);
  if (btnZoomOut) btnZoomOut.addEventListener('click', zoomOutHandler);
  if (btnResetView) btnResetView.addEventListener('click', resetHandler);

  return {
    getState: () => state,
    setState: (newState) => {
      state = Object.assign({}, state, newState);
      update();
    },
    zoom: (factor, clientX, clientY) => {
      const rect = svg.getBoundingClientRect();
      const focal = (clientX !== undefined && clientY !== undefined) ? { clientX, clientY } : null;
      state = calculateZoom(state, factor, focal, rect);
      update();
    },
    resetView: resetHandler,
    hasMovedDuringDrag: () => hasMoved,
    destroy: () => {
      svg.removeEventListener('mousedown', onMouseDown);
      window.removeEventListener('mousemove', onMouseMove);
      window.removeEventListener('mouseup', onMouseUp);
      svg.removeEventListener('wheel', onWheel);
      if (btnZoomIn) btnZoomIn.removeEventListener('click', zoomInHandler);
      if (btnZoomOut) btnZoomOut.removeEventListener('click', zoomOutHandler);
      if (btnResetView) btnResetView.removeEventListener('click', resetHandler);
    }
  };
}

if (typeof module !== 'undefined' && module.exports) {
  module.exports = {
    DEFAULT_VIEW_CONFIG,
    createViewState,
    clampScale,
    calculateZoom,
    calculatePan,
    calculateReset,
    getTransformString,
    applyTransform,
    initPanZoom
  };
}

if (typeof window !== 'undefined') {
  window.PanZoom = {
    DEFAULT_VIEW_CONFIG,
    createViewState,
    clampScale,
    calculateZoom,
    calculatePan,
    calculateReset,
    getTransformString,
    applyTransform,
    initPanZoom
  };
}
