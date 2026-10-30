const test = require('node:test');
const assert = require('node:assert/strict');
const {
  PALETTE,
  getActiveThemePalette,
  getMermaidThemeVariables,
  MERMAID_FONT_FAMILY,
  MERMAID_FONT_SIZE
} = require('./app.js');

test('PALETTE defines light warm-paper and dark calibrated-ink semantic tokens', () => {
  assert.ok(PALETTE, 'PALETTE must be exported');
  assert.deepEqual(PALETTE.light, {
    bg: '#fbf9f4',
    surface: '#ffffff',
    ink: '#2b2a33',
    muted: '#6c6a78',
    line: '#d9d4c7',
    accent: '#e4572e',
    node: '#bcd4ff',
    node2: '#ffe3a3',
    note: '#fff3b0'
  });

  assert.deepEqual(PALETTE.dark, {
    bg: '#17171c',
    surface: '#202027',
    ink: '#ececf1',
    muted: '#9a98a8',
    line: '#3a3a46',
    accent: '#ff7a59',
    node: '#3f5f94',
    node2: '#8a6a22',
    note: '#5c5320'
  });
});

test('getActiveThemePalette resolves light palette by default and dark palette on dark mode', () => {
  const defaultPalette = getActiveThemePalette();
  assert.equal(defaultPalette.bg, '#fbf9f4');
  assert.equal(defaultPalette.ink, '#2b2a33');

  const darkPalette = getActiveThemePalette('dark');
  assert.equal(darkPalette.bg, '#17171c');
  assert.equal(darkPalette.ink, '#ececf1');
});

test('getMermaidThemeVariables correctly maps active palette to Mermaid theme properties', () => {
  const lightVars = getMermaidThemeVariables(PALETTE.light);
  assert.equal(lightVars.fontFamily, MERMAID_FONT_FAMILY);
  assert.equal(lightVars.fontSize, MERMAID_FONT_SIZE);
  assert.equal(lightVars.background, '#fbf9f4');
  assert.equal(lightVars.textColor, '#2b2a33');
  assert.equal(lightVars.primaryColor, '#bcd4ff');
  assert.equal(lightVars.primaryTextColor, '#2b2a33');
  assert.equal(lightVars.primaryBorderColor, '#2b2a33');
  assert.equal(lightVars.lineColor, '#2b2a33');
  assert.equal(lightVars.clusterBkg, '#ffe3a3');
  assert.equal(lightVars.clusterBorder, '#2b2a33');
  assert.equal(lightVars.edgeLabelBackground, '#ffffff');
  assert.equal(lightVars.labelTextColor, '#2b2a33');

  const darkVars = getMermaidThemeVariables(PALETTE.dark);
  assert.equal(darkVars.background, '#17171c');
  assert.equal(darkVars.textColor, '#ececf1');
  assert.equal(darkVars.primaryColor, '#3f5f94');
  assert.equal(darkVars.primaryTextColor, '#ececf1');
  assert.equal(darkVars.primaryBorderColor, '#ececf1');
  assert.equal(darkVars.lineColor, '#ececf1');
  assert.equal(darkVars.clusterBkg, '#8a6a22');
  assert.equal(darkVars.clusterBorder, '#ececf1');
  assert.equal(darkVars.edgeLabelBackground, '#202027');
  assert.equal(darkVars.labelTextColor, '#ececf1');
});

const { mermaidConfig, DEFAULT_OPTIONS } = require('./app.js');

test('mermaidConfig sets up handDrawn look, deterministic seed, and base theme by default', () => {
  const config = mermaidConfig();
  assert.equal(config.startOnLoad, false);
  assert.equal(config.theme, 'base');
  assert.equal(config.look, 'handDrawn');
  assert.equal(config.handDrawnSeed, 7);
  assert.equal(config.fontFamily, MERMAID_FONT_FAMILY);
  assert.equal(config.fontSize, MERMAID_FONT_SIZE);
  assert.ok(config.themeVariables);
  assert.equal(config.themeVariables.background, '#fbf9f4');
  assert.equal(config.themeVariables.textColor, '#2b2a33');
});

test('mermaidConfig allows overriding look, seed, font, and custom colors', () => {
  const config = mermaidConfig({
    look: 'classic',
    seed: 99,
    font: 'sans-serif',
    fontSize: '14px',
    colors: {
      bg: '#000000',
      ink: '#ffffff',
      surface: '#111111',
      node: '#222222',
      node2: '#333333',
      note: '#444444',
      muted: '#888888'
    }
  });

  assert.equal(config.look, 'classic');
  assert.equal(config.handDrawnSeed, 99);
  assert.equal(config.fontFamily, 'sans-serif');
  assert.equal(config.fontSize, '14px');
  assert.equal(config.themeVariables.background, '#000000');
  assert.equal(config.themeVariables.textColor, '#ffffff');
  assert.equal(config.themeVariables.primaryColor, '#222222');
});

const { ensureClusterSolidFill } = require('./app.js');

test('ensureClusterSolidFill fills cluster background completely without hachure gaps', () => {
  const fillPath = {
    style: { fill: 'none', stroke: '#ffe3a3' },
    setAttribute(k, v) { this[k] = v; }
  };
  const strokePath = {
    style: { stroke: '#2b2a33' },
    setAttribute(k, v) { this[k] = v; }
  };
  const clusterEl = {
    querySelectorAll(selector) {
      if (selector === 'path') return [fillPath, strokePath];
      if (selector === 'rect') return [];
      return [];
    }
  };
  const mockSvg = {
    querySelectorAll(selector) {
      if (selector === '.cluster') return [clusterEl];
      return [];
    }
  };

  ensureClusterSolidFill(mockSvg, { node2: '#ffe3a3', ink: '#2b2a33' });

  assert.equal(fillPath.style.fill, '#ffe3a3');
  assert.equal(fillPath.style.stroke, 'none');
  assert.equal(fillPath.fill, '#ffe3a3');
  assert.equal(fillPath.stroke, 'none');
  assert.equal(strokePath.style.stroke, '#2b2a33');
  assert.equal(strokePath.stroke, '#2b2a33');
});


