// Progressive enhancement only: move focus to the error summary or the flash
// message after a submit. Every flow works without this script.
(function () {
  var el = document.getElementById('error-summary') || document.getElementById('flash');
  if (el) { el.focus(); }
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
    msg.textContent = extendable
      ? 'It ends around ' + clock(expires) + '. Choose Stay signed in to keep working.'
      : 'It ends around ' + clock(expires) + ' and cannot be extended. Finish what you are doing, then sign in again.';
    box.hidden = false;
    box.focus();
    clearTimeout(endTimer);
    endTimer = setTimeout(over, Math.max(0, expires - Date.now()));
  }
  function over() {
    extend.hidden = true;
    msg.textContent = 'Your session has ended. Sign in again to continue. Your changes on this page have not been sent.';
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
      .catch(function () { msg.textContent = 'Could not extend the session. Sign in again.'; extend.hidden = true; });
  });
  document.getElementById('session-dismiss').addEventListener('click', function () { box.hidden = true; });
  schedule();
})();
