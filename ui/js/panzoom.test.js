const test = require('node:test');
const assert = require('node:assert/strict');
const {
  createViewState,
  clampScale,
  calculateZoom,
  calculatePan,
  calculateReset,
  getTransformString
} = require('./panzoom.js');

test('createViewState returns default pan/zoom state', () => {
  const state = createViewState();
  assert.equal(state.x, 0);
  assert.equal(state.y, 0);
  assert.equal(state.scale, 1);
  assert.equal(state.minScale, 0.15);
  assert.equal(state.maxScale, 3.0);
});

test('clampScale clamps scale within minimum and maximum bounds', () => {
  assert.equal(clampScale(0.05, 0.15, 3.0), 0.15, 'should clamp to minScale');
  assert.equal(clampScale(4.5, 0.15, 3.0), 3.0, 'should clamp to maxScale');
  assert.equal(clampScale(1.5, 0.15, 3.0), 1.5, 'should preserve scale in bounds');
});

test('calculateZoom scales correctly and respects zoom bounds', () => {
  const state = createViewState();
  const zoomedIn = calculateZoom(state, 1.2);
  assert.equal(zoomedIn.scale, 1.2);

  // Zooming in many times should not exceed maxScale
  let maxed = state;
  for (let i = 0; i < 20; i++) {
    maxed = calculateZoom(maxed, 1.5);
  }
  assert.equal(maxed.scale, 3.0, 'scale must be bounded at maxScale');

  // Zooming out many times should not drop below minScale
  let mined = state;
  for (let i = 0; i < 20; i++) {
    mined = calculateZoom(mined, 0.5);
  }
  assert.equal(mined.scale, 0.15, 'scale must be bounded at minScale');
});

test('calculateZoom focal point keeps target point stationary relative to viewport', () => {
  const state = createViewState();
  const stageBounds = { left: 0, top: 0, width: 800, height: 600 };
  const focalPoint = { clientX: 400, clientY: 300 };

  const zoomed = calculateZoom(state, 2.0, focalPoint, stageBounds);
  assert.equal(zoomed.scale, 2.0);
  assert.equal(zoomed.x, -400);
  assert.equal(zoomed.y, -300);
});

test('calculatePan updates coordinates by dx and dy', () => {
  const state = createViewState();
  const panned = calculatePan(state, 50, -30);
  assert.equal(panned.x, 50);
  assert.equal(panned.y, -30);
  assert.equal(panned.scale, 1);
});

test('calculateReset restores x: 0, y: 0, scale: 1', () => {
  const state = { x: 120, y: -80, scale: 2.4, minScale: 0.15, maxScale: 3.0 };
  const reset = calculateReset(state);
  assert.equal(reset.x, 0);
  assert.equal(reset.y, 0);
  assert.equal(reset.scale, 1);
  assert.equal(reset.minScale, 0.15);
  assert.equal(reset.maxScale, 3.0);
});

test('getTransformString formats SVG transform correctly', () => {
  const state = { x: 45, y: -20, scale: 1.5 };
  assert.equal(getTransformString(state), 'translate(45, -20) scale(1.5)');
});
