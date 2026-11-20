// UX-047, PER-022, UX-030: exercise the production enhancement without a
// browser or database. The server supplies complete sentences as plain text.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const vm = require('node:vm');

const script = fs.readFileSync(path.join(__dirname, '../internal/web/static/app.js'), 'utf8');
const now = Date.UTC(2026, 10, 19, 12, 0);
const snapshot = 'Page generated <time>Zoë $& $\' $` $$</time>. Figures stay at this snapshot.';

function element(attributes = {}, textContent = '') {
  const listeners = new Map();
  return {
    hidden: true,
    textContent,
    focused: false,
    getAttribute(name) { return attributes[name] ?? null; },
    setAttribute(name, value) { attributes[name] = value; },
    addEventListener(name, callback) { listeners.set(name, callback); },
    dispatch(name) { listeners.get(name)?.(); },
    focus() { this.focused = true; },
    set innerHTML(_) { assert.fail('catalogue text must not become HTML'); },
  };
}

function run({ online = true, session = false, extendable = true, remaining = 180000, response, pathname = '/n/one/trades/new' } = {}) {
  let current = now;
  let timerID = 0;
  const timers = new Map();
  const events = new Map();
  const requests = [];
  const draft = { value: 'Unsent form values <stay> $&' };
  const csrf = { value: 'csrf-token' };
  const elements = {
    'figure-freshness': element({}, snapshot),
    'figure-stale': element({
      'data-offline-message': 'Offline catalogue sentence: {snapshot}',
      'data-history-message': 'History catalogue sentence: {snapshot}',
    }),
  };
  if (session) {
    elements['session-warning'] = element({
      'data-expires': String((now + remaining) / 1000),
      'data-extendable': String(extendable),
      'data-extendable-message': 'Continue before {time}; use the stay button.',
      'data-fixed-message': 'This fixed session ends at {time}; save your work.',
      'data-ended-message': 'The session ended; the draft remains here.',
      'data-failed-message': 'Extension failed; sign in again.',
    });
    for (const id of ['session-msg', 'session-extend', 'session-dismiss']) {
      elements[id] = element();
    }
    elements['session-login'] = element({ href: '/login' });
  }
  const window = {
    addEventListener(name, callback) { events.set(name, callback); },
  };
  const location = { pathname, search: '?member=Zo%C3%AB&keep=1' };
  Object.defineProperty(window, 'location', {
    get() { return location; },
    set() { assert.fail('warnings must not navigate away from an unsent form'); },
  });
  for (const method of ['assign', 'replace', 'reload']) {
    location[method] = () => assert.fail('warnings must not navigate');
  }
  vm.runInNewContext(script, {
    document: {
      getElementById(id) { return elements[id] ?? null; },
      querySelector(selector) { return selector === 'input[name=csrf]' ? csrf : draft; },
    },
    window,
    navigator: { onLine: online },
    Date: class extends Date { static now() { return current; } },
    setTimeout(callback, delay) { timers.set(++timerID, { callback, delay }); return timerID; },
    clearTimeout(id) { timers.delete(id); },
    fetch(url, options) {
      requests.push({ url, options });
      return response ? response() : Promise.reject(new Error('unexpected request'));
    },
  }, { filename: 'app.js' });
  return {
    elements, requests, draft, location, timers,
    dispatch(name, event = {}) { events.get(name)?.(event); },
    fireTimer(delay) {
      const matches = [...timers].filter(([, timer]) => timer.delay === delay);
      assert.equal(matches.length, 1, `expected one timer after ${delay} ms`);
      const [id, timer] = matches[0];
      timers.delete(id);
      current += delay;
      timer.callback();
    },
  };
}

function clock(time) {
  const d = new Date(time);
  return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`;
}

function assertDraftRetained(app) {
  assert.equal(app.draft.value, 'Unsent form values <stay> $&');
  assert.equal(app.location.pathname, '/n/one/trades/new');
  assert.equal(app.location.search, '?member=Zo%C3%AB&keep=1');
}

test('offline and history warnings use supplied sentences and preserve the last snapshot literally', () => {
  const app = run();
  const stale = app.elements['figure-stale'];
  assert.equal(stale.hidden, true);
  app.dispatch('offline');
  assert.equal(stale.hidden, false);
  assert.equal(stale.textContent, `Offline catalogue sentence: ${snapshot}`);
  app.dispatch('online');
  app.dispatch('pageshow', { persisted: false });
  assert.equal(stale.textContent, `Offline catalogue sentence: ${snapshot}`);
  assert.equal(stale.hidden, false);
  app.dispatch('pageshow', { persisted: true });
  assert.equal(stale.textContent, `History catalogue sentence: ${snapshot}`);
  app.dispatch('online');
  assert.equal(stale.hidden, false);
  assert.equal(app.elements['figure-freshness'].textContent, snapshot);
  assert.equal(app.requests.length, 0);
  assertDraftRetained(app);
});

test('an initially offline page keeps its server snapshot and a new page starts without a stale warning', () => {
  const app = run({ online: false });
  assert.equal(app.elements['figure-stale'].hidden, false);
  assert.equal(app.elements['figure-stale'].textContent, `Offline catalogue sentence: ${snapshot}`);
  assert.equal(app.elements['figure-freshness'].textContent, snapshot);
  assert.equal(run().elements['figure-stale'].hidden, true);
});

test('session warning appears two minutes before expiry and uses the supplied time sentence', () => {
  const app = run({ session: true });
  assert.equal(app.elements['session-warning'].hidden, true);
  app.fireTimer(60000);
  assert.equal(app.elements['session-warning'].hidden, false);
  assert.equal(app.elements['session-warning'].focused, true);
  assert.equal(app.elements['session-extend'].hidden, false);
  assert.equal(app.elements['session-msg'].textContent, `Continue before ${clock(now + 180000)}; use the stay button.`);
  assert.equal(app.elements['session-login'].getAttribute('href'), '/login?next=' + encodeURIComponent('/n/one/trades/new?member=Zo%C3%AB&keep=1'));
  app.elements['session-dismiss'].dispatch('click');
  assert.equal(app.elements['session-warning'].hidden, true);
  app.fireTimer(120000);
  assert.equal(app.elements['session-warning'].hidden, false);
  assert.equal(app.elements['session-extend'].hidden, true);
  assert.equal(app.elements['session-msg'].textContent, 'The session ended; the draft remains here.');
  assertDraftRetained(app);
});

test('fixed sessions hide extension and retain the draft when they expire', () => {
  const app = run({ session: true, extendable: false, remaining: 30000 });
  assert.equal(app.elements['session-extend'].hidden, true);
  assert.equal(app.elements['session-msg'].textContent, `This fixed session ends at ${clock(now + 30000)}; save your work.`);
  app.fireTimer(30000);
  assert.equal(app.elements['session-msg'].textContent, 'The session ended; the draft remains here.');
  assert.equal(app.requests.length, 0);
  assertDraftRetained(app);
});

test('successful extension keeps the request contract and reschedules from the server expiry', async () => {
  const app = run({
    session: true, remaining: 30000,
    response: () => Promise.resolve({ ok: true, json: () => Promise.resolve({ expires: (now + 600000) / 1000, extendable: false }) }),
  });
  app.elements['session-extend'].dispatch('click');
  await new Promise(setImmediate);
  assert.equal(app.requests.length, 1);
  assert.equal(app.requests[0].url, '/session/extend');
  assert.equal(app.requests[0].options.method, 'POST');
  assert.equal(app.requests[0].options.credentials, 'same-origin');
  assert.equal(app.requests[0].options.headers['X-CSRF-Token'], 'csrf-token');
  assert.equal(app.elements['session-warning'].hidden, true);
  assert.equal(app.timers.size, 1, 'the previous expiry timer is cancelled');
  app.fireTimer(480000);
  assert.equal(app.elements['session-warning'].hidden, false);
  assert.equal(app.elements['session-extend'].hidden, true);
  assert.equal(app.elements['session-msg'].textContent, `This fixed session ends at ${clock(now + 600000)}; save your work.`);
  assertDraftRetained(app);
});

test('failed extension uses the supplied failure message without navigation or form loss', async () => {
  for (const response of [
    () => Promise.reject(new Error('offline')),
    () => Promise.resolve({ ok: false, status: 401 }),
  ]) {
    const app = run({ session: true, remaining: 30000, response });
    app.elements['session-extend'].dispatch('click');
    await new Promise(setImmediate);
    assert.equal(app.elements['session-warning'].hidden, false);
    assert.equal(app.elements['session-extend'].hidden, true);
    assert.equal(app.elements['session-msg'].textContent, 'Extension failed; sign in again.');
    assertDraftRetained(app);
  }
});

test('session return links reject a network-path reference', () => {
  const app = run({ session: true, remaining: 30000, pathname: '//other.example/path' });
  assert.equal(app.elements['session-login'].getAttribute('href'), '/login');
});
