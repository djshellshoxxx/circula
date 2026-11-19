// Progressive enhancement only: move focus to the error summary or the flash
// message after a submit. Every flow works without this script.
(function () {
  var el = document.getElementById('error-summary') || document.getElementById('flash');
  if (el) { el.focus(); }
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
