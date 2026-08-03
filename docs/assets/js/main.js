(function() {
  // Theme toggle
  var stored = localStorage.getItem('theme');
  var prefers = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  var theme = stored || prefers;
  document.documentElement.setAttribute('data-theme', theme);

  var btn = document.getElementById('theme-toggle');
  if (btn) {
    btn.textContent = theme === 'dark' ? 'light theme' : 'dark theme';
    btn.addEventListener('click', function() {
      var cur = document.documentElement.getAttribute('data-theme');
      var next = cur === 'dark' ? 'light' : 'dark';
      document.documentElement.setAttribute('data-theme', next);
      localStorage.setItem('theme', next);
      this.textContent = next === 'dark' ? 'light theme' : 'dark theme';
    });
  }

  // Mobile nav toggle
  var navToggle = document.querySelector('.nav-toggle');
  if (navToggle) {
    navToggle.addEventListener('click', function() {
      document.querySelector('.site-header').classList.toggle('nav-open');
      var expanded = this.getAttribute('aria-expanded') === 'true';
      this.setAttribute('aria-expanded', !expanded);
    });
  }

  // Sidebar accordion
  document.querySelectorAll('.accordion-toggle, .sidebar-sub-toggle').forEach(function(toggle) {
    toggle.addEventListener('click', function() {
      var expanded = this.getAttribute('aria-expanded') === 'true';
      this.setAttribute('aria-expanded', !expanded);
      var body = this.nextElementSibling;
      while (body && !body.classList.contains('accordion-body')) {
        body = body.nextElementSibling;
      }
      if (body) {
        body.style.display = expanded ? 'none' : 'block';
      }
    });
  });

  // Active page highlight
  var currentPath = window.location.pathname;
  document.querySelectorAll('.docs-sidebar a').forEach(function(link) {
    var linkPath = link.getAttribute('href');
    if (linkPath === currentPath || linkPath === currentPath.replace('.html', '') || currentPath.endsWith(linkPath.replace('.html', ''))) {
      link.classList.add('active');
      var body = link.closest('.accordion-body');
      while (body) {
        body.style.display = 'block';
        var toggle = body.previousElementSibling;
        if (toggle && (toggle.classList.contains('accordion-toggle') || toggle.classList.contains('sidebar-sub-toggle'))) {
          toggle.setAttribute('aria-expanded', 'true');
        }
        body = body.parentElement ? body.parentElement.closest('.accordion-body') : null;
      }
    }
  });
  // Latest version pill (issue #171): fetch the newest release tag once, cache
  // it for 24h to stay well under GitHub's 60 req/hr unauthenticated limit, and
  // hide the pill on any failure so the static site degrades gracefully.
  var versionPill = document.getElementById('version-pill');
  if (versionPill) {
    var VERSION_CACHE_KEY = 'stamp_latest_version';
    var VERSION_CACHE_TTL = 24 * 60 * 60 * 1000;

    function renderVersion(tag) {
      versionPill.textContent = tag;
      versionPill.hidden = false;
    }
    function hideVersion() {
      versionPill.hidden = true;
    }

    var cached = null;
    try {
      cached = JSON.parse(localStorage.getItem(VERSION_CACHE_KEY) || 'null');
    } catch (e) {
      cached = null;
    }
    if (cached && cached.tag && Date.now() - cached.ts < VERSION_CACHE_TTL) {
      renderVersion(cached.tag);
    } else {
      fetch('https://api.github.com/repos/rossijonas/stamp/releases/latest')
        .then(function(res) {
          if (!res.ok) { throw new Error('bad status ' + res.status); }
          return res.json();
        })
        .then(function(data) {
          if (!data || typeof data.tag_name !== 'string') { throw new Error('unexpected payload'); }
          renderVersion(data.tag_name);
          try {
            localStorage.setItem(VERSION_CACHE_KEY, JSON.stringify({ tag: data.tag_name, ts: Date.now() }));
          } catch (e) { /* storage unavailable — ignore */ }
        })
        .catch(hideVersion);
    }
  }
})();
