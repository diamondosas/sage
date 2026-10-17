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
 * Ensures root namespaces, dark slate canvas background, and typography styles are injected.
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

  const clone = svg.cloneNode(true);

  const rawXml = serializer.serializeToString(clone);
  const finalSvgString = prepareStandaloneSvgString(rawXml, config);

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

/**
 * Binds export button click listener.
 */
function initExportButton(buttonElement, svgGetter, options = {}) {
  const btn = buttonElement || document.getElementById('btn-export-svg');
  if (!btn) return null;

  const handler = () => {
    const svgEl = typeof svgGetter === 'function' ? svgGetter() : svgGetter;
    exportDiagramAsSvg(svgEl, options);
  };

  btn.addEventListener('click', handler);

  return {
    destroy: () => btn.removeEventListener('click', handler)
  };
}

if (typeof module !== 'undefined' && module.exports) {
  module.exports = {
    DEFAULT_EXPORT_CONFIG,
    prepareStandaloneSvgString,
    exportDiagramAsSvg,
    initExportButton
  };
}

if (typeof window !== 'undefined') {
  window.SvgExporter = {
    DEFAULT_EXPORT_CONFIG,
    prepareStandaloneSvgString,
    exportDiagramAsSvg,
    initExportButton
  };
}
