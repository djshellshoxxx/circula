// Progressive enhancement only: move focus to the error summary or the flash
// message after a submit. Every flow works without this script.
(function () {
  var el = document.getElementById('error-summary') || document.getElementById('flash');
  if (el) { el.focus(); }
})();

// UX-028: announce ordinary page loads without replacing navigation or form
// submission. Successful controls, native validation and submitter values stay
// under browser control; attachment forms explicitly opt out.
(function () {
  var status = document.getElementById('page-loading');
  var main = document.getElementById('main');
  if (!status || !main) { return; }
  var pending = null;
  function reset() {
    clearTimeout(pending);
    pending = null;
    status.textContent = '';
    main.removeAttribute('aria-busy');
  }
  function localURL(href) {
    try {
      var url = new URL(href || window.location.href, window.location.href);
      return (url.protocol === 'http:' || url.protocol === 'https:') && url.origin === window.location.origin ? url : null;
    } catch (_) { return null; }
  }
  function sameWindow(target) { return !target || target.toLowerCase() === '_self'; }
  function announce(event, message) {
    clearTimeout(pending);
    // Another listener may cancel this event for an in-place interaction.
    // Wait until all listeners have had that opportunity.
    pending = setTimeout(function () {
      pending = null;
      if (event.defaultPrevented) { return; }
      status.textContent = status.getAttribute(message);
      main.setAttribute('aria-busy', 'true');
    }, 0);
  }
  document.addEventListener('click', function (event) {
    if (event.defaultPrevented || event.button !== 0 || event.ctrlKey || event.metaKey || event.shiftKey || event.altKey) { return; }
    var link = event.target.closest && event.target.closest('a[href]');
    if (!link || link.hasAttribute('download') || link.closest('[data-loading-ignore]') || !sameWindow(link.getAttribute('target'))) { return; }
    var url = localURL(link.getAttribute('href'));
    if (!url || (url.href.indexOf('#') !== -1 && url.pathname === window.location.pathname && url.search === window.location.search)) { return; }
    announce(event, 'data-navigation-message');
  });
  document.addEventListener('submit', function (event) {
    var form = event.target;
    var submitter = event.submitter;
    if (event.defaultPrevented || form.tagName !== 'FORM' || form.closest('[data-loading-ignore]') || (submitter && submitter.hasAttribute('data-loading-ignore'))) { return; }
    function option(name) {
      return submitter && submitter.hasAttribute('form' + name) ? submitter.getAttribute('form' + name) : form.getAttribute(name);
    }
    var method = (option('method') || 'get').toLowerCase();
    if ((method !== 'get' && method !== 'post') || !sameWindow(option('target')) || !localURL(option('action'))) { return; }
    announce(event, method === 'get' ? 'data-navigation-message' : 'data-submission-message');
  });
  window.addEventListener('pageshow', reset);
})();

// PER-022: retain the last server timestamp when connectivity fails or a saved
// page returns from browser history. Coming online is not proof of a refresh.
(function () {
  var freshness = document.getElementById('figure-freshness');
  var stale = document.getElementById('figure-stale');
  if (!freshness || !stale) { return; }
  function show(message) {
    // Function replacement preserves literal dollar sequences in the snapshot.
    stale.textContent = message.replace('{snapshot}', function () { return freshness.textContent; });
    stale.hidden = false;
  }
  window.addEventListener('offline', function () {
    show(stale.getAttribute('data-offline-message'));
  });
  window.addEventListener('pageshow', function (event) {
    if (event.persisted) { show(stale.getAttribute('data-history-message')); }
  });
  if (navigator.onLine === false) {
    show(stale.getAttribute('data-offline-message'));
  }
})();

// Session expiry warning (UX-030, WCAG 2.2.1): two minutes before the session
// is expected to end, offer to extend it. Nothing is redirected automatically,
// so unsent form values are not lost.
(function () {
  var box = document.getElementById('session-warning');
  if (!box) { return; }
  var msg = document.getElementById('session-msg');
  var extend = document.getElementById('session-extend');
  var login = document.getElementById('session-login');
  var expires = parseInt(box.getAttribute('data-expires'), 10) * 1000;
  var extendable = box.getAttribute('data-extendable') === 'true';
  var lead = 120000, warnTimer = null, endTimer = null;

  function clock(ms) {
    var d = new Date(ms);
    return ('0' + d.getHours()).slice(-2) + ':' + ('0' + d.getMinutes()).slice(-2);
  }
  function next() {
    var p = window.location.pathname + window.location.search;
    if (p.charAt(0) === '/' && p.charAt(1) !== '/') { login.setAttribute('href', '/login?next=' + encodeURIComponent(p)); }
  }
  function show() {
    next();
    extend.hidden = !extendable;
    msg.textContent = box.getAttribute(extendable ? 'data-extendable-message' : 'data-fixed-message')
      .replace('{time}', function () { return clock(expires); });
    box.hidden = false;
    box.focus();
    clearTimeout(endTimer);
    endTimer = setTimeout(over, Math.max(0, expires - Date.now()));
  }
  function over() {
    extend.hidden = true;
    msg.textContent = box.getAttribute('data-ended-message');
    box.hidden = false;
  }
  function schedule() {
    clearTimeout(warnTimer);
    var ms = expires - Date.now() - lead;
    if (ms <= 0) { show(); } else { warnTimer = setTimeout(show, Math.min(ms, 2147483647)); }
  }
  function token() {
    var i = document.querySelector('input[name=csrf]');
    return i ? i.value : '';
  }
  extend.addEventListener('click', function () {
    fetch('/session/extend', { method: 'POST', credentials: 'same-origin', headers: { 'X-CSRF-Token': token() } })
      .then(function (r) { if (!r.ok) { throw new Error('status ' + r.status); } return r.json(); })
      .then(function (j) {
        expires = j.expires * 1000; extendable = !!j.extendable;
        clearTimeout(endTimer);
        box.hidden = true; schedule();
      })
      .catch(function () { msg.textContent = box.getAttribute('data-failed-message'); extend.hidden = true; });
  });
  document.getElementById('session-dismiss').addEventListener('click', function () { box.hidden = true; });
  schedule();
})();
