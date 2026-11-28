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

function run({ online = true, session = false, extendable = true, remaining = 180000, response, expiryText, pathname = '/n/one/trades/new' } = {}) {
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
      'data-expires-text': expiryText,
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

// UX-028: exercise native navigation events without replacing the browser's
// submission, validation, target-window or download behavior.
function loadingElement(tagName, attributes = {}) {
  const attrs = { ...attributes };
  return {
    tagName: tagName.toUpperCase(),
    textContent: '',
    getAttribute(name) { return attrs[name] ?? null; },
    hasAttribute(name) { return Object.hasOwn(attrs, name); },
    setAttribute(name, value) { attrs[name] = value; },
    removeAttribute(name) { delete attrs[name]; },
    closest(selector) {
      if (selector === 'a[href]') return this.tagName === 'A' && this.hasAttribute('href') ? this : null;
      if (selector === '[data-loading-ignore]') return this.hasAttribute('data-loading-ignore') ? this : null;
      assert.fail(`unexpected selector ${selector}`);
    },
    focus() { assert.fail('loading feedback must not move focus'); },
    submit() { assert.fail('loading feedback must not replace native submission'); },
    checkValidity() { assert.fail('loading feedback must not add a new validation gate'); },
    set disabled(_) { assert.fail('successful controls must remain enabled'); },
    set innerHTML(_) { assert.fail('loading feedback must remain plain text'); },
  };
}

function runLoading() {
  const documentEvents = new Map();
  const windowEvents = new Map();
  const timers = new Map();
  let timerID = 0;
  const status = loadingElement('p', {
    'data-navigation-message': 'Loading catalogue text <next> $&',
    'data-submission-message': 'Sending catalogue text <request> $&',
  });
  const main = loadingElement('main');
  const location = new URL('https://circula.test/n/town/trades?status=proposed');
  const listen = (map, name, callback) => map.set(name, [...(map.get(name) || []), callback]);
  vm.runInNewContext(script, {
    document: {
      getElementById(id) { return { 'page-loading': status, main }[id] || null; },
      addEventListener(name, callback) { listen(documentEvents, name, callback); },
    },
    window: { location, addEventListener(name, callback) { listen(windowEvents, name, callback); } },
    navigator: { onLine: true },
    URL,
    setTimeout(callback) { timers.set(++timerID, callback); return timerID; },
    clearTimeout(id) { timers.delete(id); },
  }, { filename: 'app.js' });
  return {
    status, main,
    dispatch(name, event = {}) {
      const input = { button: 0, defaultPrevented: false, ...event };
      input.preventDefault = () => assert.fail('loading feedback must not cancel an event');
      for (const callback of documentEvents.get(name) || []) callback(input);
      return input;
    },
    windowEvent(name, event = {}) {
      for (const callback of windowEvents.get(name) || []) callback(event);
    },
    finishEvent() {
      for (const [id, callback] of [...timers]) {
        timers.delete(id);
        callback();
      }
    },
  };
}

function assertIdle(app) {
  assert.equal(app.status.textContent, '');
  assert.equal(app.main.getAttribute('aria-busy'), null);
}

test('loading announces a same-window page navigation, including keyboard link activation', () => {
  for (const detail of [0, 1]) {
    const app = runLoading();
    const link = loadingElement('a', { href: '/n/town/history' });
    app.dispatch('click', { target: link, detail });
    app.finishEvent();
    assert.equal(app.status.textContent, 'Loading catalogue text <next> $&');
    assert.equal(app.main.getAttribute('aria-busy'), 'true');
  }
});

test('native navigation feedback is synchronous even when the document blocks timer tasks', () => {
  for (const [name, target, message] of [
    ['click', loadingElement('a', { href: '/n/town/history' }), 'Loading catalogue text <next> $&'],
    ['submit', loadingElement('form', { method: 'post', action: '/n/town/trades/new' }), 'Sending catalogue text <request> $&'],
  ]) {
    const app = runLoading();
    // A native navigation can suspend timeout callbacks until the next
    // document completes. Do not advance any timers before these assertions.
    app.dispatch(name, { target });
    assert.equal(app.status.textContent, message);
    assert.equal(app.main.getAttribute('aria-busy'), 'true');
    app.finishEvent();
    assert.equal(app.status.textContent, message);
  }
});

test('loading preserves form fields and the chosen submitter name and value', () => {
  const app = runLoading();
  const form = loadingElement('form', { method: 'post', action: '/n/town/trades/new' });
  const confirm = loadingElement('button', { name: 'step', value: 'confirm' });
  const fields = Object.freeze({ amount: '12.50', csrf: 'form-token', idem: 'retry-key', note: 'Unsent words' });
  form.elements = fields;
  app.dispatch('submit', { target: form, submitter: confirm });
  app.finishEvent();
  assert.equal(app.status.textContent, 'Sending catalogue text <request> $&');
  assert.equal(app.main.getAttribute('aria-busy'), 'true');
  assert.equal(form.elements, fields);
  assert.equal(confirm.getAttribute('name'), 'step');
  assert.equal(confirm.getAttribute('value'), 'confirm');
});

test('GET filters and submitter overrides retain their native method and target', () => {
  const app = runLoading();
  const form = loadingElement('form', { method: 'post', action: '/n/town/trades/new', target: '_blank' });
  const submitter = loadingElement('button', { formmethod: 'get', formaction: '/n/town/history', formtarget: '_self' });
  app.dispatch('submit', { target: form, submitter });
  app.finishEvent();
  assert.equal(app.status.textContent, 'Loading catalogue text <next> $&');
  assert.equal(form.getAttribute('method'), 'post');
  assert.equal(form.getAttribute('target'), '_blank');
  assert.equal(submitter.getAttribute('formmethod'), 'get');
});

test('loading ignores fragments, downloads, other destinations and modified clicks', () => {
  for (const attributes of [
    { href: '#' },
    { href: '#main' },
    { href: '/n/town/trades?status=proposed#main' },
    { href: 'https://circula.test/n/town/trades?status=proposed#' },
    { href: '/n/town/history', download: '' },
    { href: '/n/town/history', target: '_blank' },
    { href: 'https://other.example/history' },
    { href: 'mailto:member@example.org' },
    { href: '/n/town/history', 'data-loading-ignore': '' },
  ]) {
    const app = runLoading();
    app.dispatch('click', { target: loadingElement('a', attributes) });
    app.finishEvent();
    assertIdle(app);
  }
  for (const override of [{ button: 1 }, { ctrlKey: true }, { metaKey: true }, { shiftKey: true }, { altKey: true }, { defaultPrevented: true }]) {
    const app = runLoading();
    app.dispatch('click', { target: loadingElement('a', { href: '/n/town/history' }), ...override });
    app.finishEvent();
    assertIdle(app);
  }
});

test('loading ignores download forms, dialog forms, other windows and submitter exclusions', () => {
  for (const [attributes, buttonAttributes] of [
    [{ method: 'post', action: '/n/town/export', 'data-loading-ignore': '' }, {}],
    [{ method: 'dialog' }, {}],
    [{ method: 'post', target: '_blank' }, {}],
    [{ method: 'post', action: 'https://other.example/submit' }, {}],
    [{ method: 'post' }, { 'data-loading-ignore': '' }],
    [{ method: 'post' }, { formtarget: '_blank' }],
    [{ method: 'post' }, { formmethod: 'dialog' }],
    [{ method: 'post' }, { formaction: 'https://other.example/submit' }],
  ]) {
    const app = runLoading();
    app.dispatch('submit', { target: loadingElement('form', attributes), submitter: loadingElement('button', buttonAttributes) });
    app.finishEvent();
    assertIdle(app);
  }
});

test('failed native validation and non-submitting async buttons leave the page idle', () => {
  const app = runLoading();
  // Native validation prevents the submit event: a button click alone must
  // never claim that a request is being sent or invoke another validity check.
  app.dispatch('click', { target: loadingElement('button', { type: 'submit' }) });
  app.finishEvent();
  assertIdle(app);
  app.dispatch('click', { target: loadingElement('button', { type: 'button', id: 'session-extend' }) });
  app.finishEvent();
  assertIdle(app);
});

test('an in-place handler can cancel a link or form after the loading listener', () => {
  for (const name of ['click', 'submit']) {
    const app = runLoading();
    const target = name === 'click' ? loadingElement('a', { href: '/n/town/history' }) : loadingElement('form', { method: 'post' });
    const event = app.dispatch(name, { target });
    event.defaultPrevented = true;
    app.finishEvent();
    assertIdle(app);
  }
});

test('pageshow clears pending and displayed feedback when a history page returns', () => {
  for (const persisted of [false, true]) {
    for (const displayed of [false, true]) {
      const app = runLoading();
      app.dispatch('click', { target: loadingElement('a', { href: '/n/town/history' }) });
      if (displayed) app.finishEvent();
      app.windowEvent('pageshow', { persisted });
      app.finishEvent();
      assertIdle(app);
    }
  }
});


// UX-024, UX-047: the server's catalogue formatter owns the visible expiry;
// the browser's zone must not replace the selected account zone.
test('session warning preserves server-formatted selected-zone expiry literally', () => {
  const expiryText = "19 Nov 2026, 17:33 IST <time> $& $' $` $$";
  for (const extendable of [true, false]) {
    const app = run({ session: true, remaining: 30000, extendable, expiryText });
    assert.equal(app.elements['session-msg'].textContent, extendable
      ? `Continue before ${expiryText}; use the stay button.`
      : `This fixed session ends at ${expiryText}; save your work.`);
    assertDraftRetained(app);
  }
});

test('extension replaces the visible expiry with the new server-formatted selected-zone text', async () => {
  const expiryText = '19 Nov 2026, 17:30 IST';
  const nextText = '19 Nov 2026, 17:40 IST <later> $&';
  const app = run({
    session: true, remaining: 30000, expiryText,
    response: () => Promise.resolve({ ok: true, json: () => Promise.resolve({
      expires: (now + 600000) / 1000, extendable: false, expires_text: nextText,
    }) }),
  });
  app.elements['session-extend'].dispatch('click');
  await new Promise(setImmediate);
  assert.equal(app.elements['session-warning'].hidden, true);
  assert.equal(app.timers.size, 1);
  app.fireTimer(480000);
  assert.equal(app.elements['session-msg'].textContent, `This fixed session ends at ${nextText}; save your work.`);
  assert.equal(app.elements['session-extend'].hidden, true);
  assertDraftRetained(app);
});
