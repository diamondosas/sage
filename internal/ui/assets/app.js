/**
 * Sage Architecture Visualizer Frontend
 * Pure vanilla JavaScript with zero external runtime dependencies.
 */

(function () {
  'use strict';

  // Application State
  const state = {
    graph: null,
    clusters: [],
    edges: [],
    selectedCluster: null,
    selectedFile: null,
    searchQuery: '',
    theme: localStorage.getItem('sage-theme') || (window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'),
    viewMode: 'graph', // 'graph' or 'cluster'
    zoom: 1,
    panX: 0,
    panY: 0,
    isDragging: false,
    dragStartX: 0,
    dragStartY: 0,
  };

  // DOM Elements Cache
  const els = {
    themeToggle: document.getElementById('theme-toggle'),
    searchInput: document.getElementById('search-input'),
    searchOverlay: document.getElementById('search-results-overlay'),
    archGraph: document.getElementById('arch-graph'),
    graphSvg: document.getElementById('graph-svg'),
    graphViewport: document.getElementById('graph-viewport'),
    clusterView: document.getElementById('cluster-view'),
    detailDrawer: document.getElementById('detail-drawer'),
    drawerBackdrop: document.getElementById('drawer-backdrop'),
    drawerCloseBtn: document.getElementById('drawer-close-btn'),
    breadcrumbs: document.getElementById('breadcrumbs'),
    statClusters: document.getElementById('stat-clusters'),
    statFiles: document.getElementById('stat-files'),
    statEdges: document.getElementById('stat-edges'),
    zoomInBtn: document.getElementById('zoom-in-btn'),
    zoomOutBtn: document.getElementById('zoom-out-btn'),
    zoomResetBtn: document.getElementById('zoom-reset-btn'),
    viewGraphBtn: document.getElementById('view-graph-btn'),
  };

  // 1. Theme Initialization
  function initTheme() {
    document.documentElement.setAttribute('data-theme', state.theme);
    updateThemeIcon();

    if (els.themeToggle) {
      els.themeToggle.addEventListener('click', () => {
        state.theme = state.theme === 'dark' ? 'light' : 'dark';
        document.documentElement.setAttribute('data-theme', state.theme);
        localStorage.setItem('sage-theme', state.theme);
        updateThemeIcon();
      });
    }
  }

  function updateThemeIcon() {
    if (!els.themeToggle) return;
    const isDark = state.theme === 'dark';
    els.themeToggle.innerHTML = isDark
      ? `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="5"/><path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"/></svg>`
      : `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/></svg>`;
  }

  // 2. Fetch Architecture Graph Data
  async function loadGraphData() {
    try {
      const response = await fetch('/api/graph');
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }
      const data = await response.json();
      state.graph = data;
      state.clusters = data.clusters || [];
      state.edges = data.edges || [];

      updateStats();
      renderGraph();
    } catch (err) {
      console.error('Failed to load architecture graph:', err);
      if (els.archGraph) {
        els.archGraph.innerHTML = `<div style="padding: 40px; text-align: center; color: var(--text-muted);">
          <h3>Error loading architecture graph</h3>
          <p style="margin-top: 8px; font-size: 13px;">${escapeHtml(err.message)}</p>
        </div>`;
      }
    }
  }

  function updateStats() {
    if (els.statClusters) els.statClusters.textContent = state.clusters.length;
    if (els.statFiles) els.statFiles.textContent = state.graph ? state.graph.total_files : 0;
    if (els.statEdges) els.statEdges.textContent = state.edges.length;
  }

  // 3. Graph Rendering via SVG
  function renderGraph() {
    if (!els.graphViewport || !state.clusters.length) return;

    els.graphViewport.innerHTML = '';

    // Create defs for arrowheads
    const defs = document.createElementNS('http://www.w3.org/2000/svg', 'defs');
    defs.innerHTML = `
      <marker id="arrowhead" viewBox="0 0 10 10" refX="22" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse">
        <path d="M 0 1 L 10 5 L 0 9 z" fill="var(--edge-color)"></path>
      </marker>
      <marker id="arrowhead-active" viewBox="0 0 10 10" refX="22" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse">
        <path d="M 0 1 L 10 5 L 0 9 z" fill="var(--edge-highlight)"></path>
      </marker>
    `;
    els.graphViewport.appendChild(defs);

    // Compute layout positions for clusters in a balanced geometric circle/ellipse
    const rect = els.archGraph.getBoundingClientRect();
    const centerX = rect.width / 2 || 600;
    const centerY = rect.height / 2 || 380;
    const radiusX = Math.min(centerX - 160, 420);
    const radiusY = Math.min(centerY - 100, 260);

    const clusterCount = state.clusters.length;
    const positions = {};

    state.clusters.forEach((cluster, idx) => {
      // Golden ratio angle distribution for clean aesthetic placement
      const angle = (idx / clusterCount) * 2 * Math.PI - Math.PI / 2;
      const x = centerX + radiusX * Math.cos(angle);
      const y = centerY + radiusY * Math.sin(angle);
      positions[cluster.id] = { x, y, width: 200, height: 90 };
    });

    // Layer 1: Edges
    const edgesGroup = document.createElementNS('http://www.w3.org/2000/svg', 'g');
    edgesGroup.setAttribute('id', 'edges-layer');

    state.edges.forEach((edge, idx) => {
      const src = positions[edge.source];
      const tgt = positions[edge.target];
      if (!src || !tgt) return;

      const path = document.createElementNS('http://www.w3.org/2000/svg', 'path');
      path.setAttribute('class', 'graph-edge');
      path.setAttribute('id', `edge-${idx}`);
      path.setAttribute('marker-end', 'url(#arrowhead)');
      path.setAttribute('data-source', edge.source);
      path.setAttribute('data-target', edge.target);

      // Curved Bezier line between source and target
      const midX = (src.x + tgt.x) / 2;
      const midY = (src.y + tgt.y) / 2;
      // Slight perpendicular offset for subtle graceful curvature
      const dx = tgt.x - src.x;
      const dy = tgt.y - src.y;
      const dist = Math.sqrt(dx * dx + dy * dy);
      const curvature = 24;
      const ctrlX = midX - (dy / (dist || 1)) * curvature;
      const ctrlY = midY + (dx / (dist || 1)) * curvature;

      path.setAttribute('d', `M ${src.x} ${src.y} Q ${ctrlX} ${ctrlY} ${tgt.x} ${tgt.y}`);

      // Edge Weight Label
      const weightGroup = document.createElementNS('http://www.w3.org/2000/svg', 'g');
      weightGroup.setAttribute('class', 'edge-weight-group');

      const bg = document.createElementNS('http://www.w3.org/2000/svg', 'rect');
      bg.setAttribute('class', 'edge-label-bg');
      bg.setAttribute('x', ctrlX - 12);
      bg.setAttribute('y', ctrlY - 9);
      bg.setAttribute('width', 24);
      bg.setAttribute('height', 18);

      const text = document.createElementNS('http://www.w3.org/2000/svg', 'text');
      text.setAttribute('class', 'edge-label-text');
      text.setAttribute('x', ctrlX);
      text.setAttribute('y', ctrlY);
      text.textContent = edge.weight || '1';

      weightGroup.appendChild(bg);
      weightGroup.appendChild(text);

      edgesGroup.appendChild(path);
      edgesGroup.appendChild(weightGroup);
    });
    els.graphViewport.appendChild(edgesGroup);

    // Layer 2: Cluster Nodes
    const nodesGroup = document.createElementNS('http://www.w3.org/2000/svg', 'g');
    nodesGroup.setAttribute('id', 'nodes-layer');

    state.clusters.forEach((cluster) => {
      const pos = positions[cluster.id];
      if (!pos) return;

      const group = document.createElementNS('http://www.w3.org/2000/svg', 'g');
      group.setAttribute('class', 'cluster-node-group');
      group.setAttribute('data-cluster-id', cluster.id);
      group.setAttribute('transform', `translate(${pos.x - pos.width / 2}, ${pos.y - pos.height / 2})`);

      // Card Background Rect
      const rectEl = document.createElementNS('http://www.w3.org/2000/svg', 'rect');
      rectEl.setAttribute('class', 'cluster-card-rect');
      rectEl.setAttribute('width', pos.width);
      rectEl.setAttribute('height', pos.height);

      // Title
      const title = document.createElementNS('http://www.w3.org/2000/svg', 'text');
      title.setAttribute('x', 16);
      title.setAttribute('y', 28);
      title.setAttribute('fill', 'var(--text-primary)');
      title.setAttribute('font-size', '13');
      title.setAttribute('font-weight', '600');
      title.setAttribute('letter-spacing', '-0.02em');
      title.textContent = cluster.name || cluster.id;

      // File Count Badge Pill
      const badgeGroup = document.createElementNS('http://www.w3.org/2000/svg', 'g');
      const badgeBg = document.createElementNS('http://www.w3.org/2000/svg', 'rect');
      badgeBg.setAttribute('x', pos.width - 56);
      badgeBg.setAttribute('y', 14);
      badgeBg.setAttribute('width', 42);
      badgeBg.setAttribute('height', 18);
      badgeBg.setAttribute('rx', 4);
      badgeBg.setAttribute('fill', 'var(--bg-secondary)');
      badgeBg.setAttribute('stroke', 'var(--border-color)');

      const badgeText = document.createElementNS('http://www.w3.org/2000/svg', 'text');
      badgeText.setAttribute('x', pos.width - 35);
      badgeText.setAttribute('y', 23);
      badgeText.setAttribute('text-anchor', 'middle');
      badgeText.setAttribute('dominant-baseline', 'central');
      badgeText.setAttribute('font-family', 'var(--font-mono)');
      badgeText.setAttribute('font-size', '10');
      badgeText.setAttribute('font-weight', '600');
      badgeText.setAttribute('fill', 'var(--text-secondary)');
      badgeText.textContent = `${cluster.file_count} f`;

      badgeGroup.appendChild(badgeBg);
      badgeGroup.appendChild(badgeText);

      // Description summary
      const desc = document.createElementNS('http://www.w3.org/2000/svg', 'text');
      desc.setAttribute('x', 16);
      desc.setAttribute('y', 52);
      desc.setAttribute('fill', 'var(--text-secondary)');
      desc.setAttribute('font-size', '11');
      const truncatedDesc = (cluster.description || '').length > 28
        ? (cluster.description || '').slice(0, 26) + '...'
        : (cluster.description || 'Architectural Domain');
      desc.textContent = truncatedDesc;

      // "Click to inspect" action hint
      const hint = document.createElementNS('http://www.w3.org/2000/svg', 'text');
      hint.setAttribute('x', 16);
      hint.setAttribute('y', 74);
      hint.setAttribute('fill', 'var(--text-muted)');
      hint.setAttribute('font-size', '10');
      hint.textContent = 'Explore files →';

      group.appendChild(rectEl);
      group.appendChild(title);
      group.appendChild(badgeGroup);
      group.appendChild(desc);
      group.appendChild(hint);

      // Hover interaction to highlight connected edges
      group.addEventListener('mouseenter', () => highlightConnections(cluster.id));
      group.addEventListener('mouseleave', () => resetHighlights());

      // Click: Drill down to cluster
      group.addEventListener('click', (e) => {
        e.stopPropagation();
        openClusterView(cluster);
      });

      nodesGroup.appendChild(group);
    });

    els.graphViewport.appendChild(nodesGroup);
    applyTransform();
  }

  function highlightConnections(clusterId) {
    const edges = els.graphViewport.querySelectorAll('.graph-edge');
    edges.forEach((edge) => {
      const src = edge.getAttribute('data-source');
      const tgt = edge.getAttribute('data-target');
      if (src === clusterId || tgt === clusterId) {
        edge.classList.add('highlighted');
        edge.setAttribute('marker-end', 'url(#arrowhead-active)');
      } else {
        edge.style.opacity = '0.2';
      }
    });

    const nodes = els.graphViewport.querySelectorAll('.cluster-node-group');
    nodes.forEach((node) => {
      const id = node.getAttribute('data-cluster-id');
      if (id !== clusterId) {
        node.style.opacity = '0.4';
      }
    });
  }

  function resetHighlights() {
    const edges = els.graphViewport.querySelectorAll('.graph-edge');
    edges.forEach((edge) => {
      edge.classList.remove('highlighted');
      edge.setAttribute('marker-end', 'url(#arrowhead)');
      edge.style.opacity = '';
    });

    const nodes = els.graphViewport.querySelectorAll('.cluster-node-group');
    nodes.forEach((node) => {
      node.style.opacity = '';
    });
  }

  // 4. Pan & Zoom Canvas Controls
  function initPanZoom() {
    if (!els.archGraph || !els.graphSvg) return;

    els.archGraph.addEventListener('mousedown', (e) => {
      if (e.button !== 0) return;
      state.isDragging = true;
      state.dragStartX = e.clientX - state.panX;
      state.dragStartY = e.clientY - state.panY;
      els.archGraph.style.cursor = 'grabbing';
    });

    window.addEventListener('mousemove', (e) => {
      if (!state.isDragging) return;
      state.panX = e.clientX - state.dragStartX;
      state.panY = e.clientY - state.dragStartY;
      applyTransform();
    });

    window.addEventListener('mouseup', () => {
      state.isDragging = false;
      if (els.archGraph) els.archGraph.style.cursor = '';
    });

    els.archGraph.addEventListener('wheel', (e) => {
      e.preventDefault();
      const zoomFactor = 1.08;
      if (e.deltaY < 0) {
        state.zoom = Math.min(state.zoom * zoomFactor, 2.5);
      } else {
        state.zoom = Math.max(state.zoom / zoomFactor, 0.4);
      }
      applyTransform();
    }, { passive: false });

    if (els.zoomInBtn) {
      els.zoomInBtn.addEventListener('click', () => {
        state.zoom = Math.min(state.zoom * 1.2, 2.5);
        applyTransform();
      });
    }

    if (els.zoomOutBtn) {
      els.zoomOutBtn.addEventListener('click', () => {
        state.zoom = Math.max(state.zoom / 1.2, 0.4);
        applyTransform();
      });
    }

    if (els.zoomResetBtn) {
      els.zoomResetBtn.addEventListener('click', () => {
        state.zoom = 1;
        state.panX = 0;
        state.panY = 0;
        applyTransform();
      });
    }
  }

  function applyTransform() {
    if (els.graphViewport) {
      els.graphViewport.setAttribute(
        'transform',
        `translate(${state.panX}, ${state.panY}) scale(${state.zoom})`
      );
    }
  }

  // 5. Cluster Drill-Down View
  function openClusterView(cluster) {
    state.selectedCluster = cluster;
    state.viewMode = 'cluster';

    if (els.archGraph) els.archGraph.style.display = 'none';
    if (els.clusterView) els.clusterView.style.display = 'block';

    updateBreadcrumbs();

    // Render cluster view content
    renderClusterFiles(cluster);
  }

  function closeClusterView() {
    state.selectedCluster = null;
    state.viewMode = 'graph';

    if (els.clusterView) els.clusterView.style.display = 'none';
    if (els.archGraph) els.archGraph.style.display = 'block';

    updateBreadcrumbs();
  }

  function renderClusterFiles(cluster) {
    if (!els.clusterView) return;

    // Filter incoming and outgoing edges for this cluster
    const inbound = state.edges.filter((e) => e.target === cluster.id);
    const outbound = state.edges.filter((e) => e.source === cluster.id);

    els.clusterView.innerHTML = `
      <div class="cluster-view-header">
        <button id="cluster-back-btn" class="cluster-view-back">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M19 12H5M12 19l-7-7 7-7"/></svg>
          Back to Architecture Overview
        </button>

        <div class="cluster-title-wrap">
          <h2 class="cluster-view-title">${escapeHtml(cluster.name || cluster.id)}</h2>
          <span class="role-badge">${cluster.file_count} files</span>
        </div>
        
        <p class="cluster-view-desc">${escapeHtml(cluster.description || 'Architectural Domain')}</p>

        <div class="cluster-chips-bar">
          <div class="cluster-chip">
            <span>Dependencies:</span>
            <strong>${outbound.length} outbound</strong>
          </div>
          <div class="cluster-chip">
            <span>Dependents:</span>
            <strong>${inbound.length} inbound</strong>
          </div>
        </div>
      </div>

      <div class="files-grid">
        ${(cluster.nodes || []).map((node) => `
          <div class="file-card" data-file-path="${escapeHtml(node.path)}">
            <div class="file-card-header">
              <span class="file-path">${escapeHtml(node.path)}</span>
              <span class="role-badge">${escapeHtml(node.role || 'Component')}</span>
            </div>
            <p class="file-summary">${escapeHtml(node.summary || 'No summary recorded.')}</p>
            <div class="file-card-footer">
              <div class="file-metric">
                <span>Exports:</span>
                <span class="file-metric-val">${(node.exports || []).length}</span>
              </div>
              <div class="file-metric">
                <span>Deps:</span>
                <span class="file-metric-val">${(node.dependencies || []).length}</span>
              </div>
            </div>
          </div>
        `).join('')}
      </div>
    `;

    // Hook back button
    const backBtn = document.getElementById('cluster-back-btn');
    if (backBtn) {
      backBtn.addEventListener('click', closeClusterView);
    }

    // Hook file cards
    const cards = els.clusterView.querySelectorAll('.file-card');
    cards.forEach((card) => {
      card.addEventListener('click', () => {
        const filePath = card.getAttribute('data-file-path');
        if (filePath) openFileDrawer(filePath);
      });
    });
  }

  // 6. Breadcrumbs
  function updateBreadcrumbs() {
    if (!els.breadcrumbs) return;

    if (state.selectedCluster) {
      els.breadcrumbs.innerHTML = `
        <span class="breadcrumb-link" id="bc-root">System Architecture</span>
        <span class="breadcrumb-sep">/</span>
        <span class="breadcrumb-active">${escapeHtml(state.selectedCluster.name || state.selectedCluster.id)}</span>
      `;
      const bcRoot = document.getElementById('bc-root');
      if (bcRoot) {
        bcRoot.addEventListener('click', closeClusterView);
      }
    } else {
      els.breadcrumbs.innerHTML = `
        <span class="breadcrumb-active">System Architecture Overview</span>
      `;
    }
  }

  // 7. Detail Drawer / Sidebar (File Inspection)
  async function openFileDrawer(filePath) {
    state.selectedFile = filePath;

    if (els.detailDrawer) els.detailDrawer.classList.add('open');
    if (els.drawerBackdrop) els.drawerBackdrop.classList.add('open');

    // Show loading skeleton
    const drawerPath = document.getElementById('drawer-path');
    const drawerRole = document.getElementById('drawer-role');
    const drawerBody = document.getElementById('drawer-body');

    if (drawerPath) drawerPath.textContent = filePath;
    if (drawerRole) drawerRole.textContent = 'Loading...';
    if (drawerBody) {
      drawerBody.innerHTML = `
        <div class="drawer-loading">
          <div class="spinner"></div>
          <span>Querying knowledge base...</span>
        </div>
      `;
    }

    try {
      const response = await fetch(`/api/file?path=${encodeURIComponent(filePath)}`);
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }
      const data = await response.json();
      renderDrawerDetails(data);
    } catch (err) {
      console.error('Failed to load file details:', err);
      if (drawerBody) {
        drawerBody.innerHTML = `
          <div class="drawer-summary-box" style="border-color: #ef4444; color: #ef4444;">
            Failed to inspect file: ${escapeHtml(err.message)}
          </div>
        `;
      }
    }
  }

  function renderDrawerDetails(file) {
    const drawerPath = document.getElementById('drawer-path');
    const drawerRole = document.getElementById('drawer-role');
    const drawerBody = document.getElementById('drawer-body');

    if (drawerPath) drawerPath.textContent = file.path;
    if (drawerRole) drawerRole.textContent = file.role || 'Source File';

    if (!drawerBody) return;

    drawerBody.innerHTML = `
      <!-- Purpose Summary -->
      <div class="drawer-section">
        <span class="section-label">Purpose Summary</span>
        <div class="drawer-summary-box">
          ${escapeHtml(file.summary || 'No purpose summary recorded in knowledge graph.')}
        </div>
      </div>

      <!-- Formal Owner -->
      <div class="drawer-section">
        <span class="section-label">Formal Owner</span>
        <div class="owner-card">
          <div class="owner-avatar">@</div>
          <div class="owner-name">${escapeHtml(file.owner || 'Unassigned (Check CODEOWNERS)')}</div>
        </div>
      </div>

      <!-- Key Symbols / Exports -->
      <div class="drawer-section">
        <span class="section-label">Exported Symbols (${(file.exports || []).length})</span>
        <div class="symbols-list">
          ${(file.exports && file.exports.length > 0)
            ? file.exports.map((exp) => `<span class="code-chip">${escapeHtml(exp)}</span>`).join('')
            : '<span style="font-size: 12px; color: var(--text-muted);">None exported</span>'}
        </div>
      </div>

      <!-- Dependencies -->
      <div class="drawer-section">
        <span class="section-label">Dependencies (${(file.dependencies || []).length})</span>
        <div class="deps-list">
          ${(file.dependencies && file.dependencies.length > 0)
            ? file.dependencies.map((dep) => `<span class="code-chip">${escapeHtml(dep)}</span>`).join('')
            : '<span style="font-size: 12px; color: var(--text-muted);">None</span>'}
        </div>
      </div>

      <!-- Top Expert Contributors -->
      <div class="drawer-section">
        <span class="section-label">Top Expert Contributors</span>
        <div class="experts-list">
          ${(file.experts && file.experts.length > 0)
            ? file.experts.map((exp) => `
              <div class="expert-row">
                <span class="expert-name">${escapeHtml(exp)}</span>
              </div>
            `).join('')
            : '<span style="font-size: 12px; color: var(--text-muted);">No git expertise recorded.</span>'}
        </div>
      </div>

      <!-- Why Lineage / Decisions -->
      <div class="drawer-section">
        <span class="section-label">Why Lineage & Decision History</span>
        <div class="timeline-list">
          ${(file.why && file.why.length > 0)
            ? file.why.map((entry, idx) => `
              <div class="timeline-item ${idx === 0 ? 'active' : ''}">
                ${escapeHtml(entry)}
              </div>
            `).join('')
            : '<span style="font-size: 12px; color: var(--text-muted); padding-left: 6px;">No architectural decisions or PR history recorded.</span>'}
        </div>
      </div>
    `;
  }

  function closeFileDrawer() {
    state.selectedFile = null;
    if (els.detailDrawer) els.detailDrawer.classList.remove('open');
    if (els.drawerBackdrop) els.drawerBackdrop.classList.remove('open');
  }

  // 8. Search & Filter Bar
  function initSearch() {
    if (!els.searchInput || !els.searchOverlay) return;

    els.searchInput.addEventListener('input', (e) => {
      const q = e.target.value.trim().toLowerCase();
      if (!q) {
        els.searchOverlay.style.display = 'none';
        return;
      }
      performSearch(q);
    });

    els.searchInput.addEventListener('focus', () => {
      const q = els.searchInput.value.trim().toLowerCase();
      if (q) performSearch(q);
    });

    // Keyboard Shortcuts: '/' or 'Cmd+K' / 'Ctrl+K' focuses search
    window.addEventListener('keydown', (e) => {
      if ((e.key === '/' || ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k')) && document.activeElement !== els.searchInput) {
        e.preventDefault();
        els.searchInput.focus();
        els.searchInput.select();
      } else if (e.key === 'Escape') {
        if (els.detailDrawer && els.detailDrawer.classList.contains('open')) {
          closeFileDrawer();
        } else if (els.searchOverlay && els.searchOverlay.style.display === 'block') {
          els.searchOverlay.style.display = 'none';
          els.searchInput.blur();
        } else if (state.viewMode === 'cluster') {
          closeClusterView();
        }
      }
    });

    // Close search overlay on click outside
    document.addEventListener('click', (e) => {
      if (!els.searchInput.contains(e.target) && !els.searchOverlay.contains(e.target)) {
        els.searchOverlay.style.display = 'none';
      }
    });
  }

  function performSearch(query) {
    if (!els.searchOverlay) return;

    const matchedClusters = [];
    const matchedFiles = [];

    state.clusters.forEach((cluster) => {
      if (
        cluster.id.toLowerCase().includes(query) ||
        (cluster.name && cluster.name.toLowerCase().includes(query)) ||
        (cluster.description && cluster.description.toLowerCase().includes(query))
      ) {
        matchedClusters.push(cluster);
      }

      (cluster.nodes || []).forEach((node) => {
        if (
          node.path.toLowerCase().includes(query) ||
          (node.summary && node.summary.toLowerCase().includes(query)) ||
          (node.role && node.role.toLowerCase().includes(query)) ||
          (node.exports && node.exports.some((x) => x.toLowerCase().includes(query)))
        ) {
          matchedFiles.push({ node, cluster });
        }
      });
    });

    if (matchedClusters.length === 0 && matchedFiles.length === 0) {
      els.searchOverlay.innerHTML = `
        <div style="padding: 16px; font-size: 13px; color: var(--text-muted); text-align: center;">
          No matching architectural domains or files found.
        </div>
      `;
      els.searchOverlay.style.display = 'block';
      return;
    }

    let html = '';

    if (matchedClusters.length > 0) {
      html += `<div style="padding: 8px 12px; font-size: 11px; font-weight: 600; text-transform: uppercase; color: var(--text-muted); background: var(--bg-secondary);">Architectural Domains (${matchedClusters.length})</div>`;
      matchedClusters.slice(0, 5).forEach((c) => {
        html += `
          <div class="search-item" data-search-type="cluster" data-id="${escapeHtml(c.id)}">
            <div class="search-item-top">
              <span class="search-item-title">${escapeHtml(c.name || c.id)}</span>
              <span class="role-badge">${c.file_count} files</span>
            </div>
            <span class="search-item-meta">${escapeHtml(c.description || '')}</span>
          </div>
        `;
      });
    }

    if (matchedFiles.length > 0) {
      html += `<div style="padding: 8px 12px; font-size: 11px; font-weight: 600; text-transform: uppercase; color: var(--text-muted); background: var(--bg-secondary);">Files (${matchedFiles.length})</div>`;
      matchedFiles.slice(0, 15).forEach(({ node, cluster }) => {
        html += `
          <div class="search-item" data-search-type="file" data-path="${escapeHtml(node.path)}">
            <div class="search-item-top">
              <span class="search-item-title">${escapeHtml(node.path)}</span>
              <span class="role-badge">${escapeHtml(node.role || cluster.id)}</span>
            </div>
            <span class="search-item-meta">${escapeHtml(node.summary || '')}</span>
          </div>
        `;
      });
    }

    els.searchOverlay.innerHTML = html;
    els.searchOverlay.style.display = 'block';

    // Hook search items click
    const items = els.searchOverlay.querySelectorAll('.search-item');
    items.forEach((item) => {
      item.addEventListener('click', () => {
        const type = item.getAttribute('data-search-type');
        els.searchOverlay.style.display = 'none';
        els.searchInput.value = '';

        if (type === 'cluster') {
          const id = item.getAttribute('data-id');
          const cluster = state.clusters.find((c) => c.id === id);
          if (cluster) openClusterView(cluster);
        } else if (type === 'file') {
          const path = item.getAttribute('data-path');
          if (path) openFileDrawer(path);
        }
      });
    });
  }

  // Utility: HTML Escape
  function escapeHtml(str) {
    if (!str) return '';
    return String(str)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#039;');
  }

  // 9. Bootstrap
  function init() {
    initTheme();
    initPanZoom();
    initSearch();

    if (els.drawerCloseBtn) {
      els.drawerCloseBtn.addEventListener('click', closeFileDrawer);
    }
    if (els.drawerBackdrop) {
      els.drawerBackdrop.addEventListener('click', closeFileDrawer);
    }

    if (els.viewGraphBtn) {
      els.viewGraphBtn.addEventListener('click', closeClusterView);
    }

    loadGraphData();
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
