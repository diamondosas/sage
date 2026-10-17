const test = require('node:test');
const assert = require('node:assert/strict');
const {
  DEFAULT_EXPORT_CONFIG,
  prepareStandaloneSvgString
} = require('./export.js');

test('DEFAULT_EXPORT_CONFIG defines dark theme background and filename', () => {
  assert.equal(DEFAULT_EXPORT_CONFIG.filename, 'sage-architecture.svg');
  assert.equal(DEFAULT_EXPORT_CONFIG.backgroundColor, '#0f172a');
  assert.equal(DEFAULT_EXPORT_CONFIG.fontFamily, "'Patrick Hand', cursive, sans-serif");
});

test('prepareStandaloneSvgString ensures standard XML namespaces and background rect', () => {
  const mockSvg = `
    <svg id="sage-canvas" viewBox="0 0 1000 600">
      <g id="viewport">
        <g class="node"><rect width="100" height="40" fill="#1e293b"></rect></g>
        <g class="annot"><ellipse cx="100" cy="50" rx="60" ry="30" stroke="#ff7a59"></ellipse></g>
      </g>
    </svg>
  `.trim();

  const exported = prepareStandaloneSvgString(mockSvg, {
    backgroundColor: '#0f172a',
    filename: 'test.svg'
  });

  assert.ok(exported.includes('xmlns="http://www.w3.org/2000/svg"'), 'must include standard SVG namespace');
  assert.ok(exported.includes('xmlns:xlink="http://www.w3.org/1999/xlink"'), 'must include xlink namespace');
  assert.ok(exported.includes('<rect width="100%" height="100%" fill="#0f172a"'), 'must inject dark slate background rect');
  assert.ok(exported.includes('Patrick Hand'), 'must preserve or inject font styles');
  assert.ok(exported.includes('<ellipse cx="100" cy="50" rx="60" ry="30" stroke="#ff7a59"'), 'must preserve active annotation rings');
});
