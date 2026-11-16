// Optional progressive enhancement. No storage of credentials, HTML or balances.
const config = document.querySelector('meta[name="circula-pwa"]');
if (config && 'serviceWorker' in navigator) {
  const base = config.content;
  if (base.startsWith('/') && base.endsWith('/') && !base.includes('..') && !base.includes('//')) {
    navigator.serviceWorker.register(`${base}sw.js`, {scope:base, updateViaCache:'none'}).catch(() => {});
    const channel = typeof BroadcastChannel === 'function' ? new BroadcastChannel(`circula-pwa:${base}`) : null;
    function scrub() {
      document.querySelectorAll('[data-private]').forEach(el => {el.replaceChildren(); el.hidden = true;});
    }
    function lock() {scrub(); const notice=document.querySelector('[data-session-notice]'); if(notice){notice.textContent='Session changed. Reload and sign in to continue.';notice.hidden=false;}}
    // A later auth adapter emits this event only after successful server revocation.
    window.addEventListener('circula:session-ended', () => {lock();channel?.postMessage('ended');});
    channel?.addEventListener('message', e => {if(e.data==='ended')lock();});
    // Scrub the document before BFCache or OS resume snapshots; restored pages
    // reload from the no-store authenticated server. No old DOM is unhidden.
    window.addEventListener('pagehide', scrub);
    window.addEventListener('pageshow', e => {if(e.persisted){scrub();location.reload();}});
    document.addEventListener('visibilitychange', () => {
      if(document.visibilityState==='hidden') scrub();
      else if(document.querySelector('[data-private][hidden]')) location.reload();
    });
    // Native POST sign-out still works without JS; notify other tabs immediately
    // of the local lock, even when the network fails to revoke the server session.
    document.querySelectorAll('form[data-signout]').forEach(form => form.addEventListener('submit', () => {lock();channel?.postMessage('ended');}));
  }
}
