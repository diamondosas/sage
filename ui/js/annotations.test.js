const test = require('node:test');
const assert = require('node:assert/strict');
const {
  DEFAULT_ANNOTATION_CONFIG,
  createAnnotationState,
  toggleHighlight,
  isHighlighted,
  clearHighlights,
  extractNodeId,
  computeNodeBounds,
  createRoughRingOptions
} = require('./annotations.js');

test('Annotation state starts empty and toggles node highlight', () => {
  const state = createAnnotationState();
  assert.equal(isHighlighted(state, 'node-A'), false);

  const res1 = toggleHighlight(state, 'node-A');
  assert.equal(res1.active, true);
  assert.equal(isHighlighted(state, 'node-A'), true);

  // Clicking again removes highlight
  const res2 = toggleHighlight(state, 'node-A');
  assert.equal(res2.active, false);
  assert.equal(isHighlighted(state, 'node-A'), false);
});

test('clearHighlights clears all active highlights', () => {
  const state = createAnnotationState(['node-A', 'node-B', 'node-C']);
  assert.equal(isHighlighted(state, 'node-B'), true);
  clearHighlights(state);
  assert.equal(isHighlighted(state, 'node-A'), false);
  assert.equal(isHighlighted(state, 'node-B'), false);
  assert.equal(isHighlighted(state, 'node-C'), false);
});

test('extractNodeId extracts canonical identifier from Mermaid node elements', () => {
  const mockEl1 = { id: 'flowchart-apiService-12', getAttribute: (a) => a === 'id' ? 'flowchart-apiService-12' : null };
  assert.equal(extractNodeId(mockEl1), 'apiService');

  const mockEl2 = { id: 'state-idle-3', getAttribute: (a) => a === 'id' ? 'state-idle-3' : null };
  assert.equal(extractNodeId(mockEl2), 'idle');

  const mockEl3 = { id: 'customNode', getAttribute: (a) => a === 'data-id' ? 'customNode' : null };
  assert.equal(extractNodeId(mockEl3), 'customNode');
});

test('computeNodeBounds transforms client rect into viewport space', () => {
  const clientRect = { left: 100, top: 150, right: 300, bottom: 250 };
  const mockInverseCTM = {
    transformPoint: (pt) => ({ x: pt.x - 50, y: pt.y - 50 })
  };

  const bounds = computeNodeBounds(clientRect, mockInverseCTM);
  assert.equal(bounds.x, 50);
  assert.equal(bounds.y, 100);
  assert.equal(bounds.w, 200);
  assert.equal(bounds.h, 100);
  assert.equal(bounds.cx, 150);
  assert.equal(bounds.cy, 150);
});

test('createRoughRingOptions configures hand-drawn rough ring parameters', () => {
  const box = { w: 100, h: 50 };
  const opts = createRoughRingOptions(box, DEFAULT_ANNOTATION_CONFIG, 2);

  assert.equal(opts.stroke, DEFAULT_ANNOTATION_CONFIG.stroke);
  assert.equal(opts.strokeWidth, DEFAULT_ANNOTATION_CONFIG.strokeWidth);
  assert.equal(opts.roughness, DEFAULT_ANNOTATION_CONFIG.roughness);
  assert.equal(opts.seed, DEFAULT_ANNOTATION_CONFIG.seed + 2);
  assert.equal(opts.ringWidth, 100 + DEFAULT_ANNOTATION_CONFIG.paddingX);
  assert.equal(opts.ringHeight, 50 + DEFAULT_ANNOTATION_CONFIG.paddingY);
});
