/**
 * Sage Architecture Visualizer - Standalone SVG Export Module
 *
 * Exports the current hand-drawn sketchy diagram (including active annotations,
 * calibrated dark slate background, and typography) as a clean, self-contained SVG file.
 */

// =============================================================================
// EXPORT CONFIGURATION
// =============================================================================
const DEFAULT_EXPORT_CONFIG = {
  // Default downloaded filename
  filename: 'sage-architecture.svg',

  // Canvas background color (preserves dark slate #0f172a in external viewers)
  backgroundColor: '#0f172a',

  // Handwriting typography font family
  fontFamily: "'Patrick Hand', cursive, sans-serif"
};

/**
 * Prepares a standalone, valid XML/SVG markup string from raw SVG text or serialized DOM.
 * Ensures root namespaces, dark slate canvas background, and typography styles are present.
 */
function prepareStandaloneSvgString(rawSvg, options = {}) {
  const config = Object.assign({}, DEFAULT_EXPORT_CONFIG, options);
  let svgStr = (typeof rawSvg === 'string') ? rawSvg.trim() : '';

  // Ensure standard SVG XML namespaces
  if (!svgStr.includes('xmlns="http://www.w3.org/2000/svg"')) {
    svgStr = svgStr.replace(/<svg\b([^>]*)>/i, '<svg xmlns="http://www.w3.org/2000/svg" $1>');
  }
  if (!svgStr.includes('xmlns:xlink=')) {
    svgStr = svgStr.replace(/<svg\b([^>]*)>/i, '<svg xmlns:xlink="http://www.w3.org/1999/xlink" $1>');
  }
  if (!svgStr.includes('version=')) {
    svgStr = svgStr.replace(/<svg\b([^>]*)>/i, '<svg version="1.1" $1>');
  }

  // Inject font definition and text styling
  const styleBlock = `
    <style>
      text, tspan {
        font-family: ${config.fontFamily};
      }
    </style>
  `.trim();

  // Inject background rect so diagram is visible when opened in image viewers or presentations
  const bgRect = `<rect width="100%" height="100%" fill="${config.backgroundColor}" />`;

  // Insert background rect and style block directly after opening <svg ...> tag
  const svgOpenTagEnd = svgStr.indexOf('>');
  if (svgOpenTagEnd !== -1) {
    const before = svgStr.slice(0, svgOpenTagEnd + 1);
    const after = svgStr.slice(svgOpenTagEnd + 1);
    svgStr = `${before}\n  ${styleBlock}\n  ${bgRect}\n${after}`;
  }

  return svgStr;
}

/**
 * Normalizes an SVG DOM element for standalone export by injecting namespaces,
 * background rect, and resetting interactive pan/zoom viewport transforms.
 */
function prepareExportClone(svg, config = DEFAULT_EXPORT_CONFIG) {
  const clone = svg.cloneNode(true);

  clone.setAttribute('xmlns', 'http://www.w3.org/2000/svg');
  clone.setAttribute('xmlns:xlink', 'http://www.w3.org/1999/xlink');
  clone.setAttribute('version', '1.1');

  // Reset viewport transform so diagram exports at natural 1.0 scale without pan clipping
  const vp = clone.querySelector('#viewport');
  if (vp) {
    vp.setAttribute('transform', 'translate(0, 0) scale(1)');
  }

  // Prepend dark slate background rect
  const bgRect = document.createElementNS('http://www.w3.org/2000/svg', 'rect');
  bgRect.setAttribute('width', '100%');
  bgRect.setAttribute('height', '100%');
  bgRect.setAttribute('fill', config.backgroundColor || DEFAULT_EXPORT_CONFIG.backgroundColor);

  // Prepend typography style block
  const styleEl = document.createElementNS('http://www.w3.org/2000/svg', 'style');
  styleEl.textContent = `text, tspan { font-family: ${config.fontFamily || DEFAULT_EXPORT_CONFIG.fontFamily}; }`;

  clone.insertBefore(styleEl, clone.firstChild);
  clone.insertBefore(bgRect, clone.firstChild);

  return clone;
}

/**
 * Exports an SVG element as a standalone downloadable file.
 *
 * @param {SVGSVGElement} svg - Target SVG DOM element.
 * @param {Object} [options] - Optional custom filename and background color.
 */
function exportDiagramAsSvg(svg, options = {}) {
  if (!svg) {
    console.warn('No SVG element provided for export.');
    return;
  }

  const config = Object.assign({}, DEFAULT_EXPORT_CONFIG, options);
  const serializer = (typeof XMLSerializer !== 'undefined') ? new XMLSerializer() : null;
  if (!serializer) return;

  let finalSvgString = '';
  if (typeof document !== 'undefined' && document.createElementNS) {
    const preparedClone = prepareExportClone(svg, config);
    finalSvgString = serializer.serializeToString(preparedClone);
  } else {
    const rawXml = serializer.serializeToString(svg.cloneNode(true));
    finalSvgString = prepareStandaloneSvgString(rawXml, config);
  }

  const blob = new Blob([finalSvgString], { type: 'image/svg+xml;charset=utf-8' });
  const downloadUrl = URL.createObjectURL(blob);

  const link = document.createElement('a');
  link.href = downloadUrl;
  link.download = config.filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);

  URL.revokeObjectURL(downloadUrl);
}

if (typeof module !== 'undefined' && module.exports) {
  module.exports = {
    DEFAULT_EXPORT_CONFIG,
    prepareStandaloneSvgString,
    prepareExportClone,
    exportDiagramAsSvg
  };
}

if (typeof window !== 'undefined') {
  window.SvgExporter = {
    DEFAULT_EXPORT_CONFIG,
    prepareStandaloneSvgString,
    prepareExportClone,
    exportDiagramAsSvg
  };
}
