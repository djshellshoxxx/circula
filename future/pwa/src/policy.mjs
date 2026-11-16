// Exact same-origin admission; intentionally does not use Workbox URL aliases.
export function requestAllowed(request, origin, assets) {
  const u = new URL(request.url);
  return request.method === 'GET' && u.origin === origin && !u.search && !u.hash &&
    !request.url.includes('?') && !request.url.includes('#') && !request.headers.has('Authorization') &&
    ['style', 'script', 'image', 'font'].includes(request.destination) && assets.has(u.href);
}
export function responseAllowed(response, expectedType) {
  const cache = response.headers.get('Cache-Control') || '';
  const type = (response.headers.get('Content-Type') || '').split(';')[0].trim();
  return response.status === 200 && !response.redirected && !response.headers.has('Set-Cookie') &&
    /(?:^|,)\s*public\s*(?:,|$)/i.test(cache) && !/(?:^|,)\s*(?:private|no-store|no-cache)(?:\s|,|=|$)/i.test(cache) &&
    type === expectedType;
}
export const offlineHTML = '<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Circula unavailable</title><main><h1>Circula is unavailable</h1><p>A connection to the server is required. No trade has been queued or sent by this offline page.</p><p>Reconnect, reload, and review any action before submitting it again. No offline balance is available.</p></main></html>';
