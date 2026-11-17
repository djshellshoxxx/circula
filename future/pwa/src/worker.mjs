import {PrecacheController} from 'workbox-precaching/PrecacheController.js';
import {requestAllowed, offlineHTML} from './policy.mjs';
import {precachePlugins} from './precache.mjs';
// Injected by the Go handler only from validated build metadata and base path.
const config = __CIRCULA_CONFIG__;
const origin = self.location.origin;
const assets = new Map(config.assets.map(a => [new URL(a.url, origin).href, a]));
const controller = new PrecacheController({cacheName: config.cacheName, plugins: precachePlugins(assets)});
controller.addToCacheList(config.assets.map(a => ({url: a.url, revision: null, integrity: `sha256-${a.sha256}`})));
self.addEventListener('install', event => {event.waitUntil(controller.install(event));});
self.addEventListener('activate', event => {event.waitUntil((async () => {
  await controller.activate(event);
  const own = controller.strategy.cacheName;
  for (const name of await caches.keys()) if (name.startsWith(config.cachePrefix) && name !== own) await caches.delete(name);
  await self.clients.claim();
})());});
self.addEventListener('fetch', event => {
  const request = event.request, u = new URL(request.url);
  if (u.origin !== origin || !u.pathname.startsWith(config.base)) return;
  event.respondWith((async () => {
    if (requestAllowed(request, origin, assets)) {
      const hit = await controller.matchPrecache(request.url);
      if (hit) return hit;
      // Missing cache is a network request, not permission to store unknown bytes.
    }
    try {
      const response = await fetch(request, {cache: 'no-store'});
      if (response.type === 'opaqueredirect' || response.type === 'opaque') return response;
      const headers = new Headers(response.headers); headers.set('Cache-Control', 'no-store');
      return new Response(response.body, {status: response.status, statusText: response.statusText, headers});
    } catch {
      if (request.method === 'GET' && request.mode === 'navigate') return new Response(offlineHTML, {status:503, headers:{'Content-Type':'text/html; charset=utf-8','Cache-Control':'no-store','Content-Security-Policy':"default-src 'none'; base-uri 'none'; frame-ancestors 'none'"}});
      return Response.error();
    }
  })());
});
// No background sync, message-based fetch, POST queue, automatic reload or replay.
