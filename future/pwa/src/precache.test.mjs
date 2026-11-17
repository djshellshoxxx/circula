import test from 'node:test';
import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
import {readBounded, verifiedResponse, precachePlugins, toBase64, MAX_ASSET_BYTES} from './precache.mjs';

const sha = bytes => createHash('sha256').update(bytes).digest('base64');
const css = text => {
  const bytes = new TextEncoder().encode(text);
  return {bytes, asset: {type: 'text/css', sha256: sha(bytes), size: bytes.length}};
};
const headers = (extra = {}) => ({'Cache-Control': 'public, max-age=31536000, immutable', 'Content-Type': 'text/css', ...extra});
// A stream that produces endless chunks and records how many were pulled/cancelled.
function endless(chunk = 4096) {
  const state = {pulled: 0, cancelled: false};
  const body = new ReadableStream({
    pull(c) { state.pulled++; c.enqueue(new Uint8Array(chunk)); },
    cancel() { state.cancelled = true; },
  });
  return {state, body};
}

test('readBounded returns exact body within bound', async () => {
  const {bytes} = css('a{color:red}');
  const out = await readBounded(new Response(bytes), bytes.length);
  assert.deepEqual(out, bytes);
  assert.deepEqual(await readBounded(new Response(null), 0), new Uint8Array(0));
});

test('readBounded rejects oversize declared Content-Length without reading', async () => {
  const {state, body} = endless();
  const r = new Response(body, {headers: {'Content-Length': '999999999'}});
  assert.equal(await readBounded(r, 100), null);
  assert.ok(state.pulled <= 1, 'only the stream constructor prefetch');
  assert.equal(state.cancelled, true);
});

test('readBounded stops and cancels an unbounded stream with no Content-Length', async () => {
  const {state, body} = endless();
  assert.equal(await readBounded(new Response(body), 10_000), null);
  assert.equal(state.cancelled, true);
  assert.ok(state.pulled <= 5, `pulled ${state.pulled} chunks`);
});

test('readBounded rejects lying and malformed Content-Length', async () => {
  for (const cl of ['abc', '-1', '1e3', '', '12345678901234', '0x10']) {
    assert.equal(await readBounded(new Response('xx', {headers: {'Content-Length': cl}}), 10), null, cl);
  }
  // Declared shorter than the actual body, and longer than the actual body.
  const short = new Response(new Blob(['abcdef']).stream(), {headers: {'Content-Length': '3'}});
  assert.equal(await readBounded(short, 10), null);
  const long = new Response(new Blob(['ab']).stream(), {headers: {'Content-Length': '5'}});
  assert.equal(await readBounded(long, 10), null);
});

test('readBounded rejects invalid bounds and stream errors', async () => {
  for (const max of [-1, 1.5, NaN, MAX_ASSET_BYTES + 1, Infinity, undefined]) {
    assert.equal(await readBounded(new Response('x'), max), null, String(max));
  }
  const failing = new ReadableStream({pull(c) { c.error(new Error('boom')); }});
  assert.equal(await readBounded(new Response(failing), 100), null);
});

test('toBase64 handles inputs larger than argument spread limits', () => {
  const big = new Uint8Array(500_000).fill(65);
  assert.equal(toBase64(big), Buffer.from(big).toString('base64'));
});

test('verifiedResponse accepts exact bytes and stores canonical headers', async () => {
  const {bytes, asset} = css('body{margin:0}');
  const out = await verifiedResponse(new Response(bytes, {headers: headers()}), asset);
  assert.ok(out);
  assert.equal(out.status, 200);
  assert.equal(out.headers.get('Content-Type'), 'text/css');
  assert.deepEqual(new Uint8Array(await out.arrayBuffer()), bytes);
});

test('verifiedResponse rejects wrong hash, size, headers and bad metadata', async () => {
  const {bytes, asset} = css('body{margin:0}');
  const mk = (body = bytes, h = headers()) => new Response(body, {headers: h});
  assert.equal(await verifiedResponse(mk(new TextEncoder().encode('body{margin:1}')), asset), null);
  assert.equal(await verifiedResponse(mk(bytes.slice(0, 3)), asset), null);
  assert.equal(await verifiedResponse(mk(new Uint8Array([...bytes, 0x20])), asset), null);
  assert.equal(await verifiedResponse(mk(bytes, headers({'Cache-Control': 'private'})), asset), null);
  assert.equal(await verifiedResponse(mk(bytes, headers({'Content-Type': 'text/html'})), asset), null);
  for (const bad of [undefined, {...asset, size: undefined}, {...asset, size: -1}, {...asset, size: MAX_ASSET_BYTES + 1}, {...asset, size: 1.5}]) {
    assert.equal(await verifiedResponse(mk(), bad), null);
  }
});

test('verifiedResponse never buffers an oversized body', async () => {
  const {asset} = css('body{margin:0}');
  const {state, body} = endless(65536);
  assert.equal(await verifiedResponse(new Response(body, {headers: headers()}), asset), null);
  assert.equal(state.cancelled, true);
  assert.ok(state.pulled <= 2);
});

test('precache plugins: only catalog URLs, no credentials, bounded cache update', async () => {
  const {bytes, asset} = css('p{}');
  const url = 'https://c.test/app/static/a.0123456789abcdef.css';
  const [plugin] = precachePlugins(new Map([[url, asset]]));
  const fetched = await plugin.requestWillFetch({request: new Request(url, {credentials: 'include'})});
  assert.equal(fetched.credentials, 'omit');
  assert.equal(fetched.redirect, 'error');
  assert.equal(fetched.cache, 'reload');
  await assert.rejects(plugin.requestWillFetch({request: new Request('https://c.test/app/api/v1/balance')}));
  assert.ok(await plugin.cacheWillUpdate({request: new Request(url), response: new Response(bytes, {headers: headers()})}));
  assert.equal(await plugin.cacheWillUpdate({request: new Request('https://c.test/x.css'), response: new Response(bytes, {headers: headers()})}), null);
});
