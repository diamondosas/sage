const test = require('node:test');
const assert = require('node:assert/strict');
const { DEFAULT_SKETCH_CONFIG, sketchify, extractRoughOptions } = require('./sketchify.js');

test('Sketchify configuration has solid fill and hand-drawn tuning defaults', () => {
  assert.equal(DEFAULT_SKETCH_CONFIG.roughness, 1.3, 'roughness should be 1.3');
  assert.equal(DEFAULT_SKETCH_CONFIG.bowing, 1.5, 'bowing should be 1.5');
  assert.equal(DEFAULT_SKETCH_CONFIG.strokeWidth, 1.5, 'strokeWidth should be 1.5');
  assert.equal(DEFAULT_SKETCH_CONFIG.fillStyle, 'solid', 'fillStyle should default to solid');
  assert.equal(DEFAULT_SKETCH_CONFIG.fillWeight, 1.0, 'fillWeight should be 1.0');
  assert.equal(DEFAULT_SKETCH_CONFIG.seed, 7, 'seed should be 7');
  assert.equal(DEFAULT_SKETCH_CONFIG.clusterBkg, '#ffe3a3', 'clusterBkg should default to soft pastel');
});

test('extractRoughOptions extracts computed stroke and fill colors with solid fill default', () => {
  const mockComputedStyle = {
    stroke: 'rgb(96, 165, 250)',
    strokeWidth: '2px',
    fill: 'rgb(30, 58, 138)',
    markerEnd: 'url(#arrowhead)'
  };

  const opts = extractRoughOptions(mockComputedStyle, DEFAULT_SKETCH_CONFIG, 'rect');
  assert.equal(opts.stroke, 'rgb(96, 165, 250)');
  assert.equal(opts.fill, 'rgb(30, 58, 138)');
  assert.equal(opts.fillStyle, 'solid', 'fillStyle should be solid');
  assert.equal(opts.fillWeight, 1.0);
  assert.equal(opts.disableMultiStroke, true, 'disableMultiStroke should be true when markerEnd exists');
});

test('extractRoughOptions ignores transparent/none fills and strokes', () => {
  const mockComputedStyle = {
    stroke: 'none',
    strokeWidth: '1px',
    fill: 'transparent',
    markerEnd: 'none'
  };

  const opts = extractRoughOptions(mockComputedStyle, DEFAULT_SKETCH_CONFIG, 'rect');
  assert.equal(opts.stroke, 'none');
  assert.equal(opts.fill, undefined);
});

function detachNode(node) {
  if (node && node.parentNode && node.parentNode.children) {
    const oldIdx = node.parentNode.children.indexOf(node);
    if (oldIdx !== -1) node.parentNode.children.splice(oldIdx, 1);
  }
}

function createMockEl(tag, attrs = {}, styles = {}, children = []) {
  const el = {
    tagName: tag.toLowerCase(),
    attributes: { ...attrs },
    style: { ...styles },
    children: [],
    parentNode: null,
    getAttribute(k) { return this.attributes[k] !== undefined ? this.attributes[k] : null; },
    setAttribute(k, v) { this.attributes[k] = String(v); },
    classList: {
      contains(cls) {
        const classAttr = el.attributes['class'] || '';
        return classAttr.split(/\s+/).some(c => c.toLowerCase() === cls.toLowerCase());
      }
    },
    closest(sel) {
      const targets = sel.split(',').map(s => s.trim().toLowerCase());
      for (const t of targets) {
        if (t.startsWith('.')) {
          const cls = t.slice(1);
          if (this.classList.contains(cls)) return this;
        } else if (t.startsWith('[')) {
          const attr = t.replace(/[\[\]"]/g, '').split('=')[0];
          if (this.attributes[attr] !== undefined) return this;
        } else if (this.tagName === t) {
          return this;
        }
      }
      return this.parentNode && this.parentNode.closest ? this.parentNode.closest(sel) : null;
    },
    querySelector(sel) {
      return this.querySelectorAll(sel)[0] || null;
    },
    querySelectorAll(sel) {
      const res = [];
      const targets = sel.split(',').map(s => s.trim().toLowerCase());
      function match(n, target) {
        if (target.startsWith('.')) {
          return n.classList && n.classList.contains(target.slice(1));
        }
        return n.tagName === target;
      }
      function walk(n) {
        for (const c of n.children) {
          if (targets.some(t => match(c, t))) res.push(c);
          walk(c);
        }
      }
      walk(this);
      return res;
    },
    insertBefore(newNode, refNode) {
      detachNode(newNode);
      const idx = this.children.indexOf(refNode);
      if (idx !== -1) this.children.splice(idx, 0, newNode);
      else this.children.push(newNode);
      newNode.parentNode = this;
    },
    appendChild(newNode) {
      detachNode(newNode);
      this.children.push(newNode);
      newNode.parentNode = this;
    }
  };

  children.forEach(c => el.appendChild(c));
  return el;
}

test('sketchify replaces SVG shapes with rough equivalents with solid fills while keeping labels visible', () => {
  const calls = [];
  const mockRough = {
    rectangle(x, y, w, h, opt) {
      calls.push({ type: 'rect', x, y, w, h, opt });
      return createMockEl('g', { 'data-rough': 'rect' });
    },
    polygon(pts, opt) {
      calls.push({ type: 'polygon', pts, opt });
      return createMockEl('g', { 'data-rough': 'polygon' });
    },
    path(d, opt) {
      const p = createMockEl('path', { d });
      const g = createMockEl('g', { 'data-rough': 'path' }, {}, [p]);
      calls.push({ type: 'path', d, opt });
      return g;
    }
  };

  const rectEl = createMockEl('rect', { width: '100', height: '50', stroke: '#60a5fa', fill: '#1e3a8a' });
  const pathEl = createMockEl('path', { d: 'M0,0 L10,10', stroke: '#94a3b8', 'marker-end': 'url(#arr)' });
  const labelText = createMockEl('text', { fill: '#fff' });
  const labelGroup = createMockEl('g', { class: 'edgeLabel' }, {}, [labelText]);

  const svg = createMockEl('svg', {}, {}, [rectEl, pathEl, labelGroup]);

  sketchify(svg, {}, mockRough);

  assert.equal(calls.length, 2, 'Should process rect and path, ignoring label');
  assert.equal(rectEl.style.display, 'none', 'Original rect is hidden');
  assert.equal(calls[0].opt.fillStyle, 'solid', 'Solid fill style applied');
  assert.equal(calls[0].opt.fillWeight, 1.0, 'Solid fill weight applied');
  assert.equal(calls[1].opt.disableMultiStroke, true, 'Marker end disables multi-stroke');
});

test('sketchify renders clusters with solid tinted fills and arranges clusters behind edges and nodes', () => {
  const calls = [];
  const mockRough = {
    rectangle(x, y, w, h, opt) {
      calls.push({ type: 'rect', x, y, w, h, opt });
      return createMockEl('g', { 'data-rough': 'rect' });
    },
    path(d, opt) {
      const p = createMockEl('path', { d });
      const g = createMockEl('g', { 'data-rough': 'path' }, {}, [p]);
      calls.push({ type: 'path', d, opt });
      return g;
    }
  };

  // Build Mermaid diagram structure: clusters, edgePaths, edgeLabels, nodes
  const clusterRect = createMockEl('rect', { width: '300', height: '200', stroke: '#2b2a33', fill: '#ffe3a3' });
  const clusterLabel = createMockEl('g', { class: 'label' }, {}, [createMockEl('text')]);
  const clusterGroup = createMockEl('g', { class: 'cluster', id: 'subGraph0' }, {}, [clusterRect, clusterLabel]);
  const clustersLayer = createMockEl('g', { class: 'clusters' }, {}, [clusterGroup]);

  const edgePath = createMockEl('path', { d: 'M10,10 L50,50', stroke: '#2b2a33' });
  const edgePathsLayer = createMockEl('g', { class: 'edgePaths' }, {}, [edgePath]);

  const edgeLabel = createMockEl('g', { class: 'edgeLabel' }, {}, [createMockEl('text')]);
  const edgeLabelsLayer = createMockEl('g', { class: 'edgeLabels' }, {}, [edgeLabel]);

  const nodeRect = createMockEl('rect', { width: '80', height: '40', stroke: '#2b2a33', fill: '#bcd4ff' });
  const nodeGroup = createMockEl('g', { class: 'node', id: 'nodeA' }, {}, [nodeRect]);
  const nodesLayer = createMockEl('g', { class: 'nodes' }, {}, [nodeGroup]);

  // Notice intentionally starting with reversed order in DOM to test reordering
  const svg = createMockEl('svg', {}, {}, [nodesLayer, edgeLabelsLayer, edgePathsLayer, clustersLayer]);

  sketchify(svg, {}, mockRough);

  // Find cluster rect call
  const clusterCall = calls.find(c => c.w === 300 && c.h === 200);
  assert.ok(clusterCall, 'Cluster rect must be processed');
  assert.equal(clusterCall.opt.fillStyle, 'solid', 'Cluster must use solid fill style, never hachure');
  assert.equal(clusterCall.opt.fill, '#ffe3a3', 'Cluster must use tinted clusterBkg fill');

  // Verify DOM hierarchy is: clusters (bottom) -> edgePaths -> edgeLabels -> nodes (top)
  const childOrder = svg.children.map(c => c.attributes.class);
  assert.deepEqual(childOrder, ['clusters', 'edgePaths', 'edgeLabels', 'nodes'], 'SVG DOM layering must place clusters at bottom and nodes at top');

  // Verify cluster sketchy twin is placed before clusterLabel inside clusterGroup
  assert.equal(clusterGroup.children[0].attributes['data-sketchy'], 'true', 'Cluster sketchy twin must be before label');
});

test('sketchify handles unparseable shapes gracefully without aborting diagram processing', () => {
  const calls = [];
  const mockRough = {
    rectangle(x, y, w, h, opt) {
      calls.push({ type: 'rect', x, y, w, h });
      return createMockEl('g', { 'data-rough': 'rect' });
    },
    path(d, opt) {
      if (d === 'corrupted-syntax') {
        throw new Error('Rough.js path parse error');
      }
      calls.push({ type: 'path', d });
      return createMockEl('g', { 'data-rough': 'path' });
    }
  };

  const badPath = createMockEl('path', { d: 'corrupted-syntax', stroke: '#2b2a33' });
  const goodRect = createMockEl('rect', { width: '120', height: '60', stroke: '#2b2a33', fill: '#bcd4ff' });

  const svg = createMockEl('svg', {}, {}, [badPath, goodRect]);

  assert.doesNotThrow(() => {
    sketchify(svg, {}, mockRough);
  }, 'Should not throw on unparseable elements');

  assert.equal(calls.length, 1, 'Valid rect should still be processed after path failure');
  assert.equal(calls[0].type, 'rect');
});

test('sketchify respects clusterBkg caller override, stroke-only none mode, and reorders standalone clusters', () => {
  const calls = [];
  const mockRough = {
    rectangle(x, y, w, h, opt) {
      calls.push({ type: 'rect', x, y, w, h, opt });
      return createMockEl('g', { 'data-rough': 'rect' });
    },
    path(d, opt) {
      return createMockEl('g', { 'data-rough': 'path' });
    }
  };

  const standaloneClusterRect = createMockEl('rect', { width: '250', height: '150', stroke: '#2b2a33' });
  const standaloneClusterGroup = createMockEl('g', { class: 'cluster', id: 'standaloneCluster' }, {}, [standaloneClusterRect]);

  const edgePath = createMockEl('path', { d: 'M0,0 L20,20', stroke: '#2b2a33' });
  const edgePathsLayer = createMockEl('g', { class: 'edgePaths' }, {}, [edgePath]);

  const nodeRect = createMockEl('rect', { width: '60', height: '30', stroke: '#2b2a33', fill: '#bcd4ff' });
  const nodesLayer = createMockEl('g', { class: 'nodes' }, {}, [createMockEl('g', { class: 'node' }, {}, [nodeRect])]);

  // Diagram with standalone cluster placed after nodes
  const svg = createMockEl('svg', {}, {}, [nodesLayer, edgePathsLayer, standaloneClusterGroup]);

  // Pass custom clusterBkg override
  sketchify(svg, { clusterBkg: '#8a6a22' }, mockRough);

  const clusterCall = calls.find(c => c.w === 250 && c.h === 150);
  assert.ok(clusterCall);
  assert.equal(clusterCall.opt.fill, '#8a6a22', 'Custom clusterBkg override applied');
  assert.equal(clusterCall.opt.fillStyle, 'solid', 'Solid fill style used');

  // Verify standalone cluster group was moved before edgePaths and nodes
  assert.equal(svg.children[0], standaloneClusterGroup, 'Standalone cluster group moved to front of edgePaths/nodes');

  // Test stroke-only mode (fillStyle: 'none')
  const callsNone = [];
  const mockRoughNone = {
    rectangle(x, y, w, h, opt) {
      callsNone.push({ type: 'rect', x, y, w, h, opt });
      return createMockEl('g', { 'data-rough': 'rect' });
    }
  };

  const clusterRectNone = createMockEl('rect', { width: '100', height: '100', stroke: '#2b2a33' });
  const clusterGroupNone = createMockEl('g', { class: 'cluster' }, {}, [clusterRectNone]);
  const svgNone = createMockEl('svg', {}, {}, [clusterGroupNone]);

  sketchify(svgNone, { fillStyle: 'none' }, mockRoughNone);
  assert.equal(callsNone[0].opt.fillStyle, 'none', 'Stroke-only none mode respected for clusters');
});

test('extractRoughOptions enforces minimum 1.5px strokeWidth, disables multi-stroke for markers or sw > 6, and extracts dashed strokeLineDash', () => {
  // Test minimum stroke width clamp
  const thinStyle = {
    stroke: '#2b2a33',
    strokeWidth: '0.5px'
  };
  const thinOpts = extractRoughOptions(thinStyle, DEFAULT_SKETCH_CONFIG, 'path');
  assert.equal(thinOpts.strokeWidth, 1.5, 'Stroke width should be clamped to at least 1.5px');

  // Test disableMultiStroke when sw > 6
  const thickStyle = {
    stroke: '#2b2a33',
    strokeWidth: '8px'
  };
  const thickOpts = extractRoughOptions(thickStyle, DEFAULT_SKETCH_CONFIG, 'path');
  assert.equal(thickOpts.disableMultiStroke, true, 'Multi-stroke should be disabled when strokeWidth > 6');

  // Test dasharray parsing
  const dashStyle = {
    stroke: '#2b2a33',
    strokeDasharray: '6, 3'
  };
  const dashOpts = extractRoughOptions(dashStyle, DEFAULT_SKETCH_CONFIG, 'path');
  assert.deepEqual(dashOpts.strokeLineDash, [6, 3], 'Dasharray should parse into strokeLineDash array');

  // Test ink fallback when stroke is missing or clear on connector
  const clearStyle = {
    stroke: 'none',
    isConnector: true
  };
  const inkOpts = extractRoughOptions(clearStyle, { ...DEFAULT_SKETCH_CONFIG, ink: '#2b2a33' }, 'path');
  assert.equal(inkOpts.stroke, '#2b2a33', 'Should fall back to ink color when stroke is empty/none on connector');

  // Test connector path enforces fill: undefined
  const connectorStyle = {
    stroke: '#2b2a33',
    fill: 'none',
    isConnector: true
  };
  const connOpts = extractRoughOptions(connectorStyle, DEFAULT_SKETCH_CONFIG, 'path');
  assert.equal(connOpts.fill, undefined, 'Connector paths must have undefined fill');
});

test('sketchify extracts connector stroke colors, falls back to ink color, enforces min 1.5px stroke, and propagates marker-end/start from attributes and computed styles', () => {
  const calls = [];
  const mockRough = {
    path(d, opt) {
      const p = createMockEl('path', { d });
      const g = createMockEl('g', { 'data-rough': 'path' }, {}, [p]);
      calls.push({ type: 'path', d, opt, g });
      return g;
    }
  };

  // Connector path with class 'path flowchart-link', no fill, no stroke attribute (relies on ink fallback), and computed markerEnd
  const connectorPath = createMockEl('path', {
    class: 'path flowchart-link',
    d: 'M10,20 L100,20',
    'stroke-dasharray': '5 5'
  }, {
    stroke: '', // empty stroke to test ink fallback
    fill: 'none',
    strokeWidth: '1px',
    markerEnd: 'url(#flowchart-pointEnd)'
  });

  const edgePathsLayer = createMockEl('g', { class: 'edgePaths' }, {}, [connectorPath]);
  const svg = createMockEl('svg', {}, {}, [edgePathsLayer]);

  sketchify(svg, { ink: '#2b2a33' }, mockRough);

  assert.equal(calls.length, 1, 'Should process connector path');
  const edgeCall = calls[0];

  assert.equal(edgeCall.opt.fill, undefined, 'Connector path must not have fill');
  assert.equal(edgeCall.opt.stroke, '#2b2a33', 'Connector path stroke must fall back to ink color');
  assert.equal(edgeCall.opt.strokeWidth, 1.5, 'Connector path stroke width must be at least 1.5px');
  assert.equal(edgeCall.opt.disableMultiStroke, true, 'disableMultiStroke must be true for marked path');
  assert.deepEqual(edgeCall.opt.strokeLineDash, [5, 5], 'strokeLineDash must preserve dasharray');

  // Verify marker-end was propagated to the replacement rough path element
  const roughPath = edgeCall.g.querySelector('path');
  assert.ok(roughPath, 'Replacement group must contain a path element');
  assert.equal(roughPath.getAttribute('marker-end'), 'url(#flowchart-pointEnd)', 'marker-end attribute must be attached to rough path');
});

test('sketchify ensures open connector paths ignore browser default black fills', () => {
  const calls = [];
  const mockRough = {
    path(d, opt) {
      const p = createMockEl('path', { d });
      const g = createMockEl('g', { 'data-rough': 'path' }, {}, [p]);
      calls.push({ type: 'path', d, opt, g });
      return g;
    }
  };

  // SVG connector path without fill attribute where computedStyle default in browser is black rgb(0, 0, 0)
  const openConnector = createMockEl('path', {
    class: 'path',
    d: 'M0,0 C20,20 40,20 60,0',
    stroke: '#2b2a33'
  }, {
    fill: 'rgb(0, 0, 0)', // browser default user-agent style
    stroke: '#2b2a33'
  });

  const svg = createMockEl('svg', {}, {}, [openConnector]);

  sketchify(svg, {}, mockRough);

  assert.equal(calls.length, 1);
  assert.equal(calls[0].opt.fill, undefined, 'Open connector path without fill attribute must ignore browser default fill');
});

test('sketchify preserves solid fills on node path shapes (e.g. cylinder or custom shapes)', () => {
  const calls = [];
  const mockRough = {
    path(d, opt) {
      const p = createMockEl('path', { d });
      const g = createMockEl('g', { 'data-rough': 'path' }, {}, [p]);
      calls.push({ type: 'path', d, opt });
      return g;
    }
  };

  // Node path (e.g. cylinder or icon) inside a node group with computed CSS fill
  const nodePath = createMockEl('path', {
    d: 'M0,0 C10,5 20,5 30,0'
  }, {
    fill: 'rgb(188, 212, 255)',
    stroke: '#2b2a33'
  });
  const nodeGroup = createMockEl('g', { class: 'node default', id: 'dbNode' }, {}, [nodePath]);
  const nodesLayer = createMockEl('g', { class: 'nodes' }, {}, [nodeGroup]);
  const svg = createMockEl('svg', {}, {}, [nodesLayer]);

  sketchify(svg, {}, mockRough);

  assert.equal(calls.length, 1, 'Node path shape must be processed');
  assert.equal(calls[0].opt.fill, 'rgb(188, 212, 255)', 'Node path shape must preserve solid fill');
  assert.equal(calls[0].opt.fillStyle, 'solid', 'Node path must use solid fill style');
});

test('sketchify resolves ink stroke color from CSS variable --ink in dark mode', () => {
  const calls = [];
  const mockRough = {
    path(d, opt) {
      const p = createMockEl('path', { d });
      const g = createMockEl('g', { 'data-rough': 'path' }, {}, [p]);
      calls.push({ type: 'path', d, opt });
      return g;
    }
  };

  // Mock global window and document with dark mode --ink CSS variable
  const originalWindow = global.window;
  const originalDoc = global.document;

  global.document = {
    documentElement: {}
  };
  global.window = {
    getComputedStyle(el) {
      return {
        getPropertyValue(prop) {
          if (prop === '--ink') return '#ececf1';
          return '';
        }
      };
    }
  };

  try {
    const connectorPath = createMockEl('path', {
      class: 'path flowchart-link',
      d: 'M0,0 L50,50'
    }, {
      stroke: '', // empty to trigger ink fallback
      fill: 'none'
    });
    const edgePathsLayer = createMockEl('g', { class: 'edgePaths' }, {}, [connectorPath]);
    const svg = createMockEl('svg', {}, {}, [edgePathsLayer]);

    sketchify(svg, {}, mockRough);

    assert.equal(calls.length, 1);
    assert.equal(calls[0].opt.stroke, '#ececf1', 'Connector path must resolve dark mode ink color from CSS variable');
  } finally {
    global.window = originalWindow;
    global.document = originalDoc;
  }
});


