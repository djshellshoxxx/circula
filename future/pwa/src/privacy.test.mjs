import test from 'node:test';
import assert from 'node:assert/strict';
import {initPrivacy, registerWorker, validBase, SESSION_NOTICE} from './privacy.mjs';

// Minimal DOM double: just what the lifecycle code touches.
function fakeEnv(overrides = {}) {
  const listeners = {window: {}, document: {}};
  const mk = which => (type, fn) => { (listeners[which][type] ||= []).push(fn); };
  const el = () => ({children: ['secret'], hidden: false, attrs: {}, textContent: '',
    replaceChildren() { this.children = []; }, setAttribute(k, v) { this.attrs[k] = v; }});
  const privates = [el(), el()];
  const notice = el(); notice.hidden = true;
  const forms = [{listeners: {}, addEventListener(t, f) { this.listeners[t] = f; }}];
  const doc = {
    visibilityState: 'visible',
    addEventListener: mk('document'),
    querySelectorAll(sel) { return sel === '[data-private]' ? privates : sel === 'form[data-signout]' ? forms : []; },
    querySelector(sel) {
      if (sel === '[data-session-notice]') return notice;
      if (sel === '[data-scrubbed]') return privates.find(p => 'data-scrubbed' in p.attrs) || null;
      return null;
    },
  };
  const state = {reloads: 0, posted: [], channels: []};
  class FakeChannel {
    constructor(name) { this.name = name; this.listeners = {}; state.channels.push(this); }
    postMessage(m) { state.posted.push(m); }
    addEventListener(t, f) { this.listeners[t] = f; }
  }
  const env = {document: doc, window: {addEventListener: mk('window')}, location: {reload() { state.reloads++; }},
    BroadcastChannel: FakeChannel, base: '/app/', ...overrides};
  return {env, listeners, privates, notice, forms, state, doc};
}
const fire = (l, which, type, ev = {}) => (l[which][type] || []).forEach(f => f(ev));
const scrubbed = privates => privates.every(p => p.children.length === 0 && p.hidden && 'data-scrubbed' in p.attrs);

test('pagehide, freeze and hidden visibility scrub private DOM', () => {
  for (const trigger of [l => fire(l, 'window', 'pagehide'), l => fire(l, 'document', 'freeze'), (l, d) => { d.visibilityState = 'hidden'; fire(l, 'document', 'visibilitychange'); }]) {
    const f = fakeEnv(); initPrivacy(f.env); trigger(f.listeners, f.doc);
    assert.ok(scrubbed(f.privates));
  }
});

test('lifecycle works without service worker, BroadcastChannel, or a valid base', () => {
  for (const patch of [{BroadcastChannel: undefined}, {BroadcastChannel: class { constructor() { throw new Error('denied'); } }}, {base: undefined}, {base: '../evil'}, {BroadcastChannel: null}]) {
    const f = fakeEnv(patch); // no navigator at all: lifecycle never touches it
    initPrivacy(f.env);
    fire(f.listeners, 'window', 'pagehide');
    assert.ok(scrubbed(f.privates), JSON.stringify(Object.keys(patch)));
    fire(f.listeners, 'window', 'circula:session-ended');
    assert.equal(f.notice.textContent, SESSION_NOTICE);
    assert.equal(f.notice.hidden, false);
    f.forms[0].listeners.submit();
  }
});

test('a throwing channel does not disable other handlers', () => {
  const f = fakeEnv({BroadcastChannel: class { postMessage() { throw new Error('x'); } addEventListener() { throw new Error('y'); } }});
  initPrivacy(f.env);
  fire(f.listeners, 'window', 'circula:session-ended');
  assert.ok(scrubbed(f.privates));
  fire(f.listeners, 'window', 'pagehide');
  assert.ok(scrubbed(f.privates));
});

test('restored (persisted) pages scrub and reload; fresh loads do not', () => {
  const f = fakeEnv(); initPrivacy(f.env);
  fire(f.listeners, 'window', 'pageshow', {persisted: false});
  assert.equal(f.state.reloads, 0);
  assert.ok(!scrubbed(f.privates));
  fire(f.listeners, 'window', 'pageshow', {persisted: true});
  assert.equal(f.state.reloads, 1);
  assert.ok(scrubbed(f.privates));
});

test('returning visible after a scrub reloads; untouched page does not', () => {
  const f = fakeEnv(); initPrivacy(f.env);
  fire(f.listeners, 'document', 'visibilitychange');
  assert.equal(f.state.reloads, 0);
  f.doc.visibilityState = 'hidden'; fire(f.listeners, 'document', 'visibilitychange');
  f.doc.visibilityState = 'visible'; fire(f.listeners, 'document', 'visibilitychange');
  assert.equal(f.state.reloads, 1);
  assert.ok(scrubbed(f.privates), 'old DOM is never unhidden');
});

test('sign-out and session-ended lock locally and notify other tabs; peers lock on message', () => {
  const f = fakeEnv(); const {channel} = initPrivacy(f.env);
  assert.equal(channel.name, 'circula-pwa:/app/');
  f.forms[0].listeners.submit();
  assert.deepEqual(f.state.posted, ['ended']);
  assert.ok(scrubbed(f.privates));
  const g = fakeEnv(); const {channel: c2} = initPrivacy(g.env);
  c2.listeners.message({data: 'other'});
  assert.ok(!scrubbed(g.privates));
  assert.equal(g.notice.hidden, true);
  c2.listeners.message({data: 'ended'});
  assert.ok(scrubbed(g.privates));
  assert.equal(g.notice.textContent, SESSION_NOTICE);
  assert.deepEqual(g.state.posted, [], 'receiving a lock does not rebroadcast');
});

test('channel carries no account data', () => {
  const f = fakeEnv(); initPrivacy(f.env);
  fire(f.listeners, 'window', 'circula:session-ended');
  assert.deepEqual(f.state.posted, ['ended']);
});

test('validBase and registerWorker are strict and optional', async () => {
  for (const b of ['/', '/app/', '/a/b/']) assert.equal(validBase(b), true, b);
  for (const b of ['app/', '/app', '/a/../b/', '//evil/', 'https://x/', '', null, undefined, 5]) assert.equal(validBase(b), false, String(b));
  assert.equal(registerWorker({navigator: undefined}, '/app/'), undefined);
  assert.equal(registerWorker({navigator: {}}, '/app/'), undefined);
  const calls = [];
  const navigator = {serviceWorker: {register: (...a) => { calls.push(a); return Promise.reject(new Error('no')); }}};
  await registerWorker({navigator}, '/app/');
  assert.deepEqual(calls, [['/app/sw.js', {scope: '/app/', updateViaCache: 'none'}]]);
  assert.equal(registerWorker({navigator}, '/../x/'), undefined);
  assert.equal(calls.length, 1);
  assert.equal(registerWorker({navigator: {serviceWorker: {register() { throw new Error('sync'); }}}}, '/app/'), undefined);
});
