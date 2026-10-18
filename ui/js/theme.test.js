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
