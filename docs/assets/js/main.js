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

  // Sidebar accordion
  document.querySelectorAll('.accordion-toggle').forEach(function(toggle) {
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
      if (body) {
        body.style.display = 'block';
        var toggle = body.previousElementSibling;
        if (toggle && toggle.classList.contains('accordion-toggle')) {
          toggle.setAttribute('aria-expanded', 'true');
        }
      }
    }
  });
})();
