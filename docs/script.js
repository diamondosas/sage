/**
 * Sage Documentation JavaScript
 * Enhances documentation with copy buttons, instant search, responsive sidebar, and active link tracking.
 */

document.addEventListener('DOMContentLoaded', () => {
  // ── 1. Setup Code Copy Buttons ──────────────────────────────
  const codeBlocks = document.querySelectorAll('.docs-code-block');
  
  codeBlocks.forEach(block => {
    // If block doesn't already have a button in a header
    let copyBtn = block.querySelector('.copy-code-btn');
    if (!copyBtn) {
      copyBtn = document.createElement('button');
      copyBtn.type = 'button';
      copyBtn.className = 'copy-code-btn';
      copyBtn.innerHTML = `
        <svg width="13" height="13" viewBox="0 0 16 16" fill="currentColor">
          <path d="M4 1.5H3a2 2 0 0 0-2 2V14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V3.5a2 2 0 0 0-2-2h-1v1h1a1 1 0 0 1 1 1V14a1 1 0 0 1-1 1H3a1 1 0 0 1-1-1V3.5a1 1 0 0 1 1-1h1v-1z"/>
          <path d="M9.5 1a.5.5 0 0 1 .5.5v1a.5.5 0 0 1-.5.5h-3a.5.5 0 0 1-.5-.5v-1a.5.5 0 0 1 .5-.5h3zm-3-1A1.5 1.5 0 0 0 5 1.5v1A1.5 1.5 0 0 0 6.5 4h3A1.5 1.5 0 0 0 11 2.5v-1A1.5 1.5 0 0 0 9.5 0h-3z"/>
        </svg>
        <span>Copy</span>
      `;
      block.appendChild(copyBtn);
    }

    copyBtn.addEventListener('click', async (e) => {
      e.preventDefault();
      const codeElement = block.querySelector('pre code');
      if (!codeElement) return;

      const textToCopy = codeElement.innerText;
      try {
        await navigator.clipboard.writeText(textToCopy);
        copyBtn.classList.add('copied');
        const span = copyBtn.querySelector('span');
        const originalText = span ? span.textContent : 'Copy';
        if (span) span.textContent = 'Copied!';

        setTimeout(() => {
          copyBtn.classList.remove('copied');
          if (span) span.textContent = originalText;
        }, 2000);
      } catch (err) {
        console.error('Failed to copy to clipboard:', err);
      }
    });
  });

  // ── 2. Sidebar Toggler for Mobile ───────────────────────────
  const sidebarToggler = document.getElementById('docs-sidebar-toggler');
  const sidebar = document.getElementById('docs-sidebar');

  if (sidebarToggler && sidebar) {
    sidebarToggler.addEventListener('click', () => {
      if (sidebar.classList.contains('sidebar-visible')) {
        sidebar.classList.remove('sidebar-visible');
        sidebar.classList.add('sidebar-hidden');
      } else {
        sidebar.classList.remove('sidebar-hidden');
        sidebar.classList.add('sidebar-visible');
      }
    });

    // Close sidebar on link click in mobile view
    const navLinks = sidebar.querySelectorAll('.nav-link');
    navLinks.forEach(link => {
      link.addEventListener('click', () => {
        if (window.innerWidth < 1200) {
          sidebar.classList.remove('sidebar-visible');
          sidebar.classList.add('sidebar-hidden');
        }
      });
    });
  }

  // ── 3. Anchor Links Generator ──────────────────────────────
  const headings = document.querySelectorAll('h1.docs-heading, h2.section-heading, h3.section-heading');
  headings.forEach(heading => {
    const parent = heading.closest('.docs-section, .docs-article');
    if (parent && parent.id) {
      const anchor = document.createElement('a');
      anchor.className = 'anchorjs-link';
      anchor.href = `#${parent.id}`;
      anchor.setAttribute('aria-label', 'Anchor');
      anchor.innerHTML = '#';
      heading.appendChild(anchor);
    }
  });

  // ── 4. Instant Fuzzy / Keyword Search ───────────────────────
  const searchInput = document.querySelector('.search-input');
  const searchResults = document.getElementById('results');

  if (searchInput && searchResults) {
    // Build search index from headings and descriptions
    const searchIndex = [];
    headings.forEach(heading => {
      const parent = heading.closest('.docs-section, .docs-article');
      if (parent && parent.id) {
        const text = heading.textContent.replace('#', '').trim();
        searchIndex.push({
          id: parent.id,
          title: text,
          category: parent.classList.contains('docs-article') ? 'Section' : 'Topic'
        });
      }
    });

    searchInput.addEventListener('input', (e) => {
      const query = e.target.value.toLowerCase().trim();
      if (!query || query.length < 2) {
        searchResults.innerHTML = '';
        return;
      }

      const matches = searchIndex.filter(item => {
        return item.title.toLowerCase().includes(query) || item.id.toLowerCase().includes(query);
      });

      if (matches.length === 0) {
        searchResults.innerHTML = `<div class="p-3 text-muted small">No documentation matches for "${query}"</div>`;
        return;
      }

      let html = '';
      matches.slice(0, 8).forEach(item => {
        html += `
          <a href="#${item.id}">
            <strong>${item.title}</strong>
            <span>Jump to #${item.id}</span>
          </a>
        `;
      });
      searchResults.innerHTML = html;
    });

    // Close search dropdown on click outside
    document.addEventListener('click', (e) => {
      if (!searchResults.contains(e.target) && e.target !== searchInput) {
        searchResults.innerHTML = '';
      }
    });
  }

  // ── 5. Active Link Highlighting on Scroll (ScrollSpy) ────────
  const observerSections = document.querySelectorAll('.docs-section, .docs-article');
  const navLinksMap = new Map();

  document.querySelectorAll('.docs-nav .nav-link').forEach(link => {
    const href = link.getAttribute('href');
    if (href && href.startsWith('#')) {
      navLinksMap.set(href.slice(1), link);
    }
  });

  const observerOptions = {
    root: null,
    rootMargin: '-80px 0px -60% 0px',
    threshold: 0
  };

  const observer = new IntersectionObserver((entries) => {
    entries.forEach(entry => {
      if (entry.isIntersecting) {
        const id = entry.target.id;
        const activeLink = navLinksMap.get(id);
        if (activeLink) {
          document.querySelectorAll('.docs-nav .nav-link').forEach(l => l.classList.remove('active'));
          activeLink.classList.add('active');
        }
      }
    });
  }, observerOptions);

  observerSections.forEach(section => observer.observe(section));

  // ── 6. Highlight.js init if present ─────────────────────────
  if (window.hljs) {
    window.hljs.highlightAll();
  }
});
