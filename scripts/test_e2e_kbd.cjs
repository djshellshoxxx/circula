// Exercise the actual keyboard helper with a small page/DOM adapter. The
// colliding link/checkbox names and their order are from PR #99's captured
// session-confirmation accessibility tree (artifact 14533534257).
// This verifies helper targeting and key sequences, not browser rendering or
// native keyboard behavior; the real Playwright suite remains a separate gate.
const assert = require('node:assert/strict');
const test = require('node:test');
const vm = require('node:vm');
const { kbActivate, kbSubmit, tabTo } = require('../e2e/kbd');

function element(tagName, text = '', attributes = {}, label = '') {
  return {
    tagName,
    type: attributes.type || '',
    innerText: text,
    labels: label ? [{ innerText: label }] : [],
    value: attributes.value || '',
    checked: false,
    getAttribute(name) { return attributes[name] ?? null; },
  };
}

function keyboardPage(elements) {
  const document = { body: {}, activeElement: null };
  document.activeElement = document.body;
  const context = vm.createContext({ document });
  const presses = [];
  const activations = [];
  let at = -1;
  let navigation;
  const page = {
    // Execute the helper's own browser callback, rather than implementing its
    // name matching a second time in this adapter.
    async evaluate(callback, argument) {
      context.argument = argument;
      return vm.runInContext(`(${callback.toString()})(argument)`, context);
    },
    keyboard: {
      async press(key) {
        if (key === 'Tab') {
          at = (at + 1) % elements.length;
          document.activeElement = elements[at];
        } else if (key === 'Space') {
          const el = document.activeElement;
          if (el.tagName === 'INPUT' && el.type === 'checkbox') el.checked = !el.checked;
        } else if (key === 'Enter') {
          activations.push(document.activeElement);
          if (navigation) navigation();
        } else {
          assert.fail(`unexpected key ${key}`);
        }
        presses.push({ key, element: document.activeElement });
      },
      async type(text) { document.activeElement.value += text; },
    },
    waitForNavigation() { return new Promise(resolve => { navigation = resolve; }); },
    click() { assert.fail('helper must retain keyboard-only interaction'); },
    fill() { assert.fail('helper must retain keyboard-only interaction'); },
    focus() { assert.fail('helper must navigate focus with Tab'); },
  };
  return { page, presses, activations };
}

for (const label of ['I want to end this session.', 'I want to end all my other sessions.']) {
  test(`checkbox confirmation skips the identically named error link: ${label}`, async () => {
    const errorLink = element('A', label, { href: '#f-confirm' });
    const checkbox = element('INPUT', '', { type: 'checkbox', id: 'f-confirm', name: 'confirm', value: 'yes' }, label);
    const submit = element('BUTTON', 'End session', { type: 'submit' });
    const { page, presses, activations } = keyboardPage([errorLink, checkbox, submit]);
    await kbSubmit(page, [[new RegExp(`^${label.replace('.', '\\.')}$`), true]], /^End session$/);
    assert.equal(checkbox.checked, true, 'the confirmation checkbox must be checked');
    assert.deepEqual(presses.filter(p => p.key === 'Space').map(p => p.element), [checkbox]);
    assert.deepEqual(activations, [submit]);
    assert.deepEqual(presses.map(p => p.key), ['Tab', 'Tab', 'Space', 'Tab', 'Enter']);
  });
}

test('a checkbox without an error summary retains ordinary Tab/Space/Enter behavior', async () => {
  const checkbox = element('INPUT', '', { type: 'checkbox' }, 'I understand');
  const submit = element('BUTTON', 'Continue');
  const { page, presses, activations } = keyboardPage([checkbox, submit]);
  await kbSubmit(page, [[/^I understand$/, true]], /^Continue$/);
  assert.equal(checkbox.checked, true);
  assert.deepEqual(presses.map(p => p.key), ['Tab', 'Space', 'Tab', 'Enter']);
  assert.deepEqual(activations, [submit]);
});

test('a same-name link without the required checkbox is rejected before submission', async () => {
  const errorLink = element('A', 'I understand', { href: '#missing' });
  const submit = element('BUTTON', 'Continue');
  const { page, presses, activations } = keyboardPage([errorLink, submit]);
  await assert.rejects(kbSubmit(page, [[/^I understand$/, true]], /^Continue$/), /could not reach/);
  assert.equal(presses.length, 80, 'retain the bounded keyboard search');
  assert.ok(presses.every(p => p.key === 'Tab'));
  assert.deepEqual(activations, []);
});

test('text entry and intentional link activation retain their existing behavior', async () => {
  const input = element('INPUT', '', { type: 'text' }, 'Login');
  const submit = element('BUTTON', 'Sign in');
  const form = keyboardPage([input, submit]);
  await kbSubmit(form.page, [[/^Login$/, 'example']], /^Sign in$/);
  assert.equal(input.value, 'example');
  assert.deepEqual(form.activations, [submit]);

  const link = element('A', 'Help', { href: '/help' });
  const navigation = keyboardPage([link]);
  await kbActivate(navigation.page, /^Help$/);
  assert.deepEqual(navigation.activations, [link]);
  assert.deepEqual(navigation.presses.map(p => p.key), ['Tab', 'Enter']);
});

test('ordinary tabTo callers retain their explicit search budget', async () => {
  const { page, presses } = keyboardPage([element('A', 'Unrelated')]);
  await assert.rejects(tabTo(page, /^Absent$/, 3), /within 3 presses/);
  assert.equal(presses.length, 3);
});
