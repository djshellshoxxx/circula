// Shared fixtures: seeded state, per-role signed-in pages and the axe runner.
const fs = require('fs');
const path = require('path');
const base = require('@playwright/test');

const STATE_DIR = path.join(__dirname, '.state');

function state() {
  return JSON.parse(fs.readFileSync(path.join(STATE_DIR, 'state.json'), 'utf8'));
}

const AXE_PATH = require.resolve('axe-core/axe.min.js');
// WCAG 2.0 A and AA as required, plus the 2.1 additions, which the same engine checks at no extra cost.
const AXE_TAGS = ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'];

/** Runs axe on the current page and returns the violations. */
async function axeViolations(page) {
  await page.addScriptTag({ path: AXE_PATH });
  return page.evaluate(async (tags) => {
    // eslint-disable-next-line no-undef
    const r = await axe.run(document, { runOnly: { type: 'tag', values: tags }, resultTypes: ['violations'] });
    return r.violations.map((v) => ({
      id: v.id, impact: v.impact, help: v.help, helpUrl: v.helpUrl,
      nodes: v.nodes.map((n) => ({ target: n.target.join(' '), html: n.html.slice(0, 200), summary: n.failureSummary })),
    }));
  }, AXE_TAGS);
}

/** Fails with a readable list when the current page violates any rule. */
async function expectNoAxeViolations(page, label) {
  const results = await axeViolations(page);
  if (results.length) {
    const text = results.map((v) =>
      `  [${v.impact}] ${v.id}: ${v.help}\n` +
      v.nodes.map((n) => `      ${n.target}\n      ${n.html}\n      ${(n.summary || '').replace(/\n/g, '\n      ')}`).join('\n')).join('\n');
    throw new Error(`${label}: ${results.length} axe violation(s) at ${page.url()}\n${text}`);
  }
}

const test = base.test.extend({
  st: async ({}, use) => { await use(state()); }, // eslint-disable-line no-empty-pattern
  baseURL: async ({ st }, use) => { await use(st.base); },
  /** Opens a new page signed in as the named seeded user. */
  as: async ({ browser, st }, use) => {
    const contexts = [];
    await use(async (who) => {
      const ctx = await browser.newContext({ baseURL: st.base, bypassCSP: true, storageState: path.join(STATE_DIR, who + '.json') });
      contexts.push(ctx);
      return ctx.newPage();
    });
    for (const c of contexts) await c.close();
  },
});

module.exports = { test, expect: base.expect, expectNoAxeViolations, axeViolations, state, STATE_DIR };
