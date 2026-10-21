// Progressive enhancement only: move focus to the error summary or the flash
// message after a submit. Every flow works without this script.
(function () {
  var el = document.getElementById('error-summary') || document.getElementById('flash');
  if (el) { el.focus(); }
})();
