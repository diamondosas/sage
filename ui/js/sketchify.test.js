const test = require('node:test');
const assert = require('node:assert/strict');
const { DEFAULT_SKETCH_CONFIG, sketchify, extractRoughOptions } = require('./sketchify.js');

test('Sketchify configuration has high-density hachure defaults', () => {
  assert.equal(DEFAULT_SKETCH_CONFIG.hachureGap, 3, 'hachureGap should be 3px for high-density');
  assert.equal(DEFAULT_SKETCH_CONFIG.fillWeight, 1.0, 'fillWeight should be 1.0 for crispness');
  assert.equal(DEFAULT_SKETCH_CONFIG.roughness, 1.3, 'roughness should be ~1.3');
  assert.equal(DEFAULT_SKETCH_CONFIG.bowing, 1.2, 'bowing should be ~1.2');
  assert.equal(DEFAULT_SKETCH_CONFIG.fillStyle, 'hachure', 'fillStyle should default to hachure');
});

test('extractRoughOptions extracts computed stroke and fill colors', () => {
  const mockComputedStyle = {
    stroke: 'rgb(96, 165, 250)',
    strokeWidth: '2px',
    fill: 'rgb(30, 58, 138)',
    markerEnd: 'url(#arrowhead)'
  };

  const opts = extractRoughOptions(mockComputedStyle, DEFAULT_SKETCH_CONFIG, 'rect');
  assert.equal(opts.stroke, 'rgb(96, 165, 250)');
  assert.equal(opts.fill, 'rgb(30, 58, 138)');
  assert.equal(opts.hachureGap, 3);
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

test('sketchify replaces SVG shapes with rough equivalents while keeping labels visible', () => {
  function createEl(tag, attrs = {}, styles = {}, children = []) {
    return {
      tagName: tag.toLowerCase(),
      attributes: { ...attrs },
      style: { ...styles },
      children: [...children],
      parentNode: null,
      getAttribute(k) { return this.attributes[k] || null; },
      setAttribute(k, v) { this.attributes[k] = String(v); },
      closest(sel) {
        if (sel.includes(this.tagName)) return this;
        return this.parentNode ? this.parentNode.closest(sel) : null;
      },
      querySelectorAll(sel) {
        const res = [];
        const targets = sel.split(',').map(s => s.trim().toLowerCase());
        function walk(n) {
          for (const c of n.children) {
            if (targets.includes(c.tagName)) res.push(c);
            walk(c);
          }
        }
        walk(this);
        return res;
      },
      insertBefore(newNode, refNode) {
        const idx = this.children.indexOf(refNode);
        if (idx !== -1) this.children.splice(idx, 0, newNode);
        else this.children.push(newNode);
        newNode.parentNode = this;
      }
    };
  }

  const calls = [];
  const mockRough = {
    rectangle(x, y, w, h, opt) {
      calls.push({ type: 'rect', x, y, w, h, opt });
      return createEl('g', { 'data-rough': 'rect' });
    },
    polygon(pts, opt) {
      calls.push({ type: 'polygon', pts, opt });
      return createEl('g', { 'data-rough': 'polygon' });
    },
    path(d, opt) {
      const p = createEl('path', { d });
      const g = createEl('g', { 'data-rough': 'path' }, {}, [p]);
      calls.push({ type: 'path', d, opt });
      return g;
    }
  };

  const rectEl = createEl('rect', { width: '100', height: '50', stroke: '#60a5fa', fill: '#1e3a8a' });
  const pathEl = createEl('path', { d: 'M0,0 L10,10', stroke: '#94a3b8', 'marker-end': 'url(#arr)' });
  const labelText = createEl('text', { fill: '#fff' });
  const labelGroup = createEl('g', { class: 'edgeLabel' }, {}, [labelText]);

  const svg = createEl('svg', {}, {}, [rectEl, pathEl, labelGroup]);
  rectEl.parentNode = svg;
  pathEl.parentNode = svg;
  labelGroup.parentNode = svg;

  sketchify(svg, {}, mockRough);

  assert.equal(calls.length, 2, 'Should process rect and path, ignoring label');
  assert.equal(rectEl.style.display, 'none', 'Original rect is hidden');
  assert.equal(calls[0].opt.hachureGap, 3, 'High density hachure gap applied');
  assert.equal(calls[0].opt.fillWeight, 1.0, 'High density fill weight applied');
  assert.equal(calls[1].opt.disableMultiStroke, true, 'Marker end disables multi-stroke');
});
