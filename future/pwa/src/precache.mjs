import {responseAllowed} from './policy.mjs';

// Hard ceiling for any single precached asset, independent of catalog metadata.
// The Go bundle loader and build script enforce the same limit.
export const MAX_ASSET_BYTES = 1 << 20;

// Reads a response body without ever buffering more than `max` bytes.
// Returns a Uint8Array of exactly the body, or null if the body is longer
// than `max`, the declared Content-Length is invalid/oversized/inconsistent,
// or the stream fails. The stream is cancelled as soon as the bound is hit.
export async function readBounded(response, max) {
  if (!Number.isSafeInteger(max) || max < 0 || max > MAX_ASSET_BYTES) return null;
  const declared = response.headers.get('Content-Length');
  if (declared !== null) {
    if (!/^[0-9]{1,10}$/.test(declared.trim()) || Number(declared) > max) {
      await response.body?.cancel().catch(() => {});
      return null;
    }
  }
  if (!response.body) return new Uint8Array(0);
  const reader = response.body.getReader();
  const chunks = [];
  let total = 0;
  try {
    for (;;) {
      const {done, value} = await reader.read();
      if (done) break;
      total += value.byteLength;
      if (total > max) { await reader.cancel().catch(() => {}); return null; }
      chunks.push(value);
    }
  } catch {
    return null;
  }
  const out = new Uint8Array(total);
  let at = 0;
  for (const c of chunks) { out.set(c, at); at += c.byteLength; }
  if (declared !== null && Number(declared) !== total) return null;
  return out;
}

export function toBase64(bytes) {
  let s = '';
  for (let i = 0; i < bytes.length; i += 0x8000) s += String.fromCharCode(...bytes.subarray(i, i + 0x8000));
  return btoa(s);
}

// Returns a fresh Response built from the verified bytes (so the stored bytes
// are exactly the hashed bytes), or null. `asset` is {type, sha256, size}.
export async function verifiedResponse(response, asset, subtle = crypto.subtle) {
  if (!asset || !Number.isSafeInteger(asset.size) || asset.size < 0 || asset.size > MAX_ASSET_BYTES) return null;
  if (!responseAllowed(response, asset.type)) return null;
  const bytes = await readBounded(response, asset.size);
  if (!bytes || bytes.byteLength !== asset.size) return null;
  const actual = toBase64(new Uint8Array(await subtle.digest('SHA-256', bytes)));
  if (actual !== asset.sha256) return null;
  return new Response(bytes, {status: 200, headers: {
    'Content-Type': asset.type,
    'Cache-Control': 'public, max-age=31536000, immutable',
  }});
}

// Workbox plugin set for the precache strategy; `assets` maps absolute URL -> asset.
export function precachePlugins(assets) {
  return [{
    async requestWillFetch({request}) {
      if (!assets.has(request.url)) throw new Error('Non-public precache request');
      return new Request(request, {credentials: 'omit', cache: 'reload', redirect: 'error'});
    },
    async cacheWillUpdate({request, response}) {
      return verifiedResponse(response, assets.get(request.url));
    },
  }];
}
