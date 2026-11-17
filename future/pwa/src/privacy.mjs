// Private-view lifecycle. Deliberately independent of service-worker and
// BroadcastChannel support: scrubbing must work on plain HTTP loopback, in
// browsers without workers, and when the channel constructor throws.
// `env` is injected so the behavior is unit-testable without a browser.
export const SESSION_NOTICE = 'Session changed. Reload and sign in to continue.';
const attempt = fn => { try { return fn(); } catch { return undefined; } };

export function initPrivacy(env) {
  const {document, window, location} = env;
  const base = typeof env.base === 'string' && env.base.startsWith('/') ? env.base : '/';
  const channel = typeof env.BroadcastChannel === 'function'
    ? attempt(() => new env.BroadcastChannel(`circula-pwa:${base}`)) : undefined;
  const scrub = () => attempt(() => {
    document.querySelectorAll('[data-private]').forEach(el => {
      el.replaceChildren();
      el.hidden = true;
      el.setAttribute('data-scrubbed', '');
    });
  });
  const lock = () => {
    scrub();
    attempt(() => {
      const notice = document.querySelector('[data-session-notice]');
      if (notice) { notice.textContent = SESSION_NOTICE; notice.hidden = false; }
    });
  };
  const announce = () => attempt(() => channel?.postMessage('ended'));
  const reload = () => attempt(() => location.reload());
  // A later auth adapter emits this event only after successful server revocation.
  window.addEventListener('circula:session-ended', () => { lock(); announce(); });
  attempt(() => channel?.addEventListener('message', e => { if (e.data === 'ended') lock(); }));
  // Scrub before BFCache/OS-resume snapshots; restored pages reload from the
  // no-store authenticated server and old DOM is never unhidden.
  window.addEventListener('pagehide', scrub);
  window.addEventListener('pageshow', e => { if (e.persisted) { scrub(); reload(); } });
  document.addEventListener('freeze', scrub);
  document.addEventListener('visibilitychange', () => {
    if (document.visibilityState === 'hidden') scrub();
    else if (document.querySelector('[data-scrubbed]')) reload();
  });
  // Native POST sign-out still works without JS; lock other tabs immediately
  // even if the network fails to revoke the server session.
  document.querySelectorAll('form[data-signout]').forEach(form =>
    form.addEventListener('submit', () => { lock(); announce(); }));
  return {scrub, lock, channel};
}

// Same-origin service-worker registration is optional enhancement only.
export function registerWorker(env, base) {
  const nav = env.navigator;
  if (!nav || !nav.serviceWorker || !validBase(base)) return undefined;
  return attempt(() => nav.serviceWorker.register(`${base}sw.js`, {scope: base, updateViaCache: 'none'}).catch(() => {}));
}
export function validBase(base) {
  return typeof base === 'string' && base.startsWith('/') && base.endsWith('/') && !base.includes('..') && !base.includes('//');
}
