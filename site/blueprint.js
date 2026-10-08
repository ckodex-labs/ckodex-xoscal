/* Progressive enhancement only: all content and destinations exist in HTML. */
(() => {
  'use strict';
  const root = document.documentElement;
  const views = [...document.querySelectorAll('.view')];
  const links = [...document.querySelectorAll('#nav a[data-view]')];
  const status = document.getElementById('portal-status');

  function targetFromHash() {
    let id;
    try { id = decodeURIComponent(location.hash.slice(1)); }
    catch { return views[0]; }
    const target = document.getElementById(id);
    return target?.closest('.view') || views[0];
  }

  function showTarget(focus) {
    const view = targetFromHash();
    views.forEach(item => item.classList.toggle('active', item === view));
    links.forEach(link => {
      const selected = link.getAttribute('href') === '#' + view.id;
      link.classList.toggle('active', selected);
      if (selected) link.setAttribute('aria-current', 'location');
      else link.removeAttribute('aria-current');
    });
    if (focus) {
      let target;
      try { target = document.getElementById(decodeURIComponent(location.hash.slice(1))) || view; }
      catch { target = view; }
      if (target === view) target = view.querySelector('.room-title') || view;
      if (!target.hasAttribute('tabindex')) target.setAttribute('tabindex', '-1');
      target.focus({preventScroll: true});
      target.scrollIntoView({block: 'start'});
    }
  }

  function filter(group, attribute, selector, matches) {
    const buttons = [...document.querySelectorAll(group + ' button')];
    buttons.forEach(button => {
      button.setAttribute('aria-pressed', String(button.classList.contains('active')));
      button.addEventListener('click', () => {
        buttons.forEach(item => {
          item.classList.toggle('active', item === button);
          item.setAttribute('aria-pressed', String(item === button));
        });
        const selected = button.dataset[attribute];
        const items = [...document.querySelectorAll(selector)];
        items.forEach(item => { item.hidden = selected !== 'all' && !matches(item, selected); });
        if (status) status.textContent = `${items.filter(item => !item.hidden).length} items shown`;
      });
    });
  }

  filter('#card-filter', 'filter', '#view-cartography .card-section', (item, value) => item.dataset.cat === value);
  filter('#persona-filter', 'pfilter', '#persona-grid .persona-card', (item, value) => item.dataset.pid === value);
  filter('#pipe-filter', 'pipef', '#pipe-grid .pipe-card', (item, value) => item.dataset.pcTags.split(',').includes(value));

  const themeButtons = [...document.querySelectorAll('.toggle[title="Theme"] button')];
  themeButtons.forEach(button => button.addEventListener('click', () => {
    root.setAttribute('data-theme', button.dataset.theme);
    themeButtons.forEach(item => {
      item.classList.toggle('on', item === button);
      item.setAttribute('aria-pressed', String(item === button));
    });
  }));

  document.querySelectorAll('.claims .claim').forEach(claim => {
    function select() {
      document.querySelectorAll('.claims .claim').forEach(item => {
        item.classList.toggle('sel', item === claim);
        item.setAttribute('aria-pressed', String(item === claim));
      });
    }
    claim.setAttribute('role', 'button');
    claim.setAttribute('tabindex', '0');
    claim.setAttribute('aria-pressed', String(claim.classList.contains('sel')));
    claim.addEventListener('click', select);
    claim.addEventListener('keydown', event => {
      if (event.key === 'Enter' || event.key === ' ') { event.preventDefault(); select(); }
    });
  });

  // Native anchors preserve deep links, browser history and no-JS navigation.
  window.addEventListener('hashchange', () => showTarget(true));
  showTarget(false);
  root.setAttribute('data-blueprint-enhanced', 'true');
})();
