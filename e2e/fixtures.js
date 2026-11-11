// Shared fixtures: seeded state, per-role signed-in pages and the axe runner.
const fs = require('fs');
const path = require('path');
const base = require('@playwright/test');

const STATE_DIR = path.join(__dirname, '.state');

function state() {
  return JSON.parse(fs.readFileSync(path.join(STATE_DIR, 'state.json'), 'utf8'));
}

const AXE_PATH = require.resolve('axe-core/axe.min.js');
// The WCAG 2.2 AA rule set (DEC-024, UX-054, QA-033): the 2.0 and 2.1 A and AA tags plus the 2.2 AA tag
// (target-size, 2.5.8). axe tags each rule with the version that introduced it, so all five are needed.
const AXE_TAGS = ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa', 'wcag22aa'];
// Both presentations of the stylesheet (prefers-color-scheme), so contrast is checked in light and dark (UX-057).
const SCHEMES = ['light', 'dark'];

/** Runs axe on the current page and returns the violations. */
async function axeViolations(page) {
  if (!(await page.evaluate(() => typeof window.axe !== 'undefined'))) await page.addScriptTag({ path: AXE_PATH });
  return page.evaluate(async (tags) => {
    // eslint-disable-next-line no-undef
    const r = await axe.run(document, { runOnly: { type: 'tag', values: tags }, resultTypes: ['violations'] });
    return r.violations.map((v) => ({
      id: v.id, impact: v.impact, help: v.help, helpUrl: v.helpUrl,
      nodes: v.nodes.map((n) => ({ target: n.target.join(' '), html: n.html.slice(0, 200), summary: n.failureSummary })),
    }));
  }, AXE_TAGS);
}

/** Switches the page to the colour scheme and confirms the stylesheet now sees it. */
async function useScheme(page, scheme) {
  await page.emulateMedia({ colorScheme: scheme });
  const seen = await page.evaluate(() => (matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'));
  if (seen !== scheme) throw new Error(`colour scheme ${scheme} was not applied (page reports ${seen})`);
}

/**
 * Fails with a readable list when the current page violates any rule, checked once per colour scheme on the same
 * DOM. The page is left in the light scheme, the default of every context.
 */
async function expectNoAxeViolations(page, label) {
  const failures = [];
  for (const scheme of SCHEMES) {
    await useScheme(page, scheme);
    const results = await axeViolations(page);
    if (results.length) {
      const text = results.map((v) =>
        `  [${v.impact}] ${v.id}: ${v.help}\n` +
        v.nodes.map((n) => `      ${n.target}\n      ${n.html}\n      ${(n.summary || '').replace(/\n/g, '\n      ')}`).join('\n')).join('\n');
      failures.push(`${label} (${scheme}): ${results.length} axe violation(s) at ${page.url()}\n${text}`);
    }
  }
  await useScheme(page, 'light');
  if (failures.length) throw new Error(failures.join('\n'));
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

module.exports = { test, expect: base.expect, expectNoAxeViolations, axeViolations, useScheme, state, STATE_DIR, AXE_TAGS, SCHEMES };
