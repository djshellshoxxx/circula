// Real-browser acceptance script for the favorites pages. It is a manual gate:
// it needs playwright-core (or playwright), a Chromium build, and axe-core, none
// of which are dependencies of this module.
//
//   cd future/favorites
//   FAVORITES_BROWSER_ADDR=127.0.0.1:18202 FAVORITES_BROWSER_SECONDS=120 \
//     go test -run TestServeForBrowser ./web &
//   NODE_PATH=<dir with node_modules containing playwright and axe-core> \
//     CHROMIUM=/opt/pw-browsers/chromium-1243/chrome-linux64/chrome \
//     node browser/check.mjs http://127.0.0.1:18202 /tmp/favorites-shots
//
// Automated axe-core results do NOT replace manual screen-reader qualification.
import { createRequire } from 'node:module';
import fs from 'node:fs';
const require = createRequire(import.meta.url);
const { chromium } = require('playwright');
const axeSource = fs.readFileSync(require.resolve('axe-core/axe.min.js'), 'utf8');

const base = process.argv[2];
const shots = process.argv[3] || '.';
fs.mkdirSync(shots, { recursive: true });
const host = new URL(base).hostname;
const results = [];
let failed = 0;
const check = (name, ok, detail = '') => {
  results.push({ name, ok, detail });
  if (!ok) failed++;
  console.log(`${ok ? 'PASS' : 'FAIL'} ${name}${detail ? ' :: ' + detail : ''}`);
};

const browser = await chromium.launch({ executablePath: process.env.CHROMIUM, args: ['--no-sandbox'] });
const ctxFor = async (user, opts = {}) => {
  const c = await browser.newContext({ baseURL: base, ...opts });
  await c.addCookies([{ name: 'sid', value: user, domain: host, path: '/' }]);
  return c;
};

async function axeRun(page, label) {
  await page.evaluate(axeSource);
  const r = await page.evaluate(async () => await axe.run(document, { runOnly: { type: 'tag', values: ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa', 'wcag22aa', 'best-practice'] } }));
  const v = r.violations.map(x => `${x.id}(${x.impact}) x${x.nodes.length}`);
  check(`axe ${label}: no violations (${r.passes.length} rules passed)`, v.length === 0, v.join('; '));
}

// 1. JavaScript disabled: every essential path works with plain forms.
{
  const c = await ctxFor('alice', { javaScriptEnabled: false });
  const p = await c.newPage();
  await p.goto('/favorites');
  check('no-JS: list renders three seeded partners', (await p.locator('ul.fav > li').count()) === 3);
  await p.fill('#target', 'erin');
  await p.fill('#nickname', 'Erin <b>nick</b>');
  await p.check('#pinned');
  await p.click('button:has-text("Save trading partner")');
  check('no-JS: add redirects to list with status notice', (await p.locator('[role=status]').innerText()).includes('added'));
  check('no-JS: pinned partner first', (await p.locator('ul.fav > li h3').first().innerText()).includes('Member erin'));
  check('no-JS: nickname escaped, not rendered as markup', (await p.locator('ul.fav > li').first().innerHTML()).includes('&lt;b&gt;nick'));
  await p.locator('ul.fav > li', { hasText: 'Member dave' }).getByRole('button', { name: /Move up/ }).click();
  const order = await p.locator('ul.fav > li h3').allInnerTexts();
  check('no-JS: move up works', order.map(s => s.replace('Pinned', '').trim()).join(',') === 'Member erin,Member bob,Member dave,Member carol', order.join(','));
  await p.locator('ul.fav > li', { hasText: 'Member bob' }).locator('summary').click();
  await p.locator('ul.fav > li', { hasText: 'Member bob' }).locator('input[name=nickname]').fill('Bobby');
  await p.locator('ul.fav > li', { hasText: 'Member bob' }).getByRole('button', { name: 'Save nickname' }).click();
  check('no-JS: rename via native details/summary form', (await p.content()).includes('Bobby'));
  await p.locator('ul.fav > li', { hasText: 'Member carol' }).getByRole('button', { name: /Start a proposal/ }).click();
  check('no-JS: draft page reached', /\/favorites\/drafts\/[0-9a-f]{32}$/.test(p.url()), p.url());
  check('no-JS: draft amount is blank and focusable', (await p.inputValue('#amount')) === '');
  await p.screenshot({ path: `${shots}/draft-nojs.png`, fullPage: true });
  await p.goto('/favorites');
  await p.locator('ul.fav > li', { hasText: 'Member carol' }).getByRole('button', { name: /Remove/ }).click();
  check('no-JS: remove works', (await p.locator('ul.fav > li', { hasText: 'Member carol' }).count()) === 0);
  await c.close();
}

// 2. Accessibility automation (needs JS only to run axe itself; pages ship no script).
{
  const c = await ctxFor('alice');
  const p = await c.newPage();
  await p.goto('/favorites');
  await axeRun(p, 'list');
  await p.screenshot({ path: `${shots}/list.png`, fullPage: true });
  await p.fill('#target', 'nobody');
  await p.click('button:has-text("Save trading partner")');
  check('error summary has role=alert', (await p.locator('[role=alert]').count()) === 1);
  await axeRun(p, 'list with error summary');
  await p.screenshot({ path: `${shots}/list-error.png`, fullPage: true });
  await p.goto('/favorites');
  await p.locator('ul.fav > li').first().getByRole('button', { name: /Start a proposal/ }).click();
  await axeRun(p, 'draft');
  await c.close();
}

// 3. Keyboard: skip link first, every control reachable, focus visible.
{
  const c = await ctxFor('alice');
  const p = await c.newPage();
  await p.goto('/favorites');
  await p.keyboard.press('Tab');
  check('keyboard: first Tab lands on the skip link', (await p.evaluate(() => document.activeElement.className)) === 'skip');
  // Focusable controls = those not hidden inside a closed <details>.
  const total = await p.evaluate(() => {
    const els = [...document.querySelectorAll('main a[href], main button, main input:not([type=hidden]), main summary, main textarea')]
      .filter(e => e.getClientRects().length > 0 && (e.tagName === 'SUMMARY' || !e.closest('details:not([open])')));
    els.forEach((e, i) => e.setAttribute('data-i', String(i)));
    return els.length;
  });
  const seen = new Set();
  for (let i = 0; i < total + 3; i++) {
    await p.keyboard.press('Tab');
    const idx = await p.evaluate(() => document.activeElement.getAttribute('data-i'));
    if (idx !== null) seen.add(idx);
  }
  check('keyboard: Tab reaches every visible control', seen.size === total, `${seen.size} of ${total}`);
  await p.focus('#target');
  const outline = await p.evaluate(() => { const s = getComputedStyle(document.activeElement); return s.outlineStyle + ' ' + s.outlineWidth; });
  check('keyboard: visible focus outline', /solid 3px/.test(outline), outline);
  await p.focus('#target');
  await p.keyboard.type('carol');
  await Promise.all([p.waitForNavigation(), p.keyboard.press('Enter')]); // submit with the keyboard only
  await p.waitForLoadState();
  check('keyboard: Enter in a field submits the form', (await p.locator('[role=status], [role=alert]').count()) === 1);
  await c.close();
}

// 4. Mobile: 320 CSS px (WCAG reflow), touch targets, no horizontal scroll.
{
  const c = await ctxFor('alice', { viewport: { width: 320, height: 700 }, hasTouch: true, isMobile: true });
  const p = await c.newPage();
  await p.goto('/favorites');
  const m = await p.evaluate(() => ({ sw: document.documentElement.scrollWidth, iw: window.innerWidth }));
  check('mobile 320px: no horizontal scrolling', m.sw <= m.iw, JSON.stringify(m));
  const small = await p.evaluate(() => [...document.querySelectorAll('main button, main input[type=text], main summary, main a')].filter(e => { const r = e.getBoundingClientRect(); return r.width > 0 && (r.height < 44 && e.tagName !== 'A' || r.height < 24); }).map(e => e.tagName + ':' + e.textContent.trim().slice(0, 20)));
  check('mobile: controls are at least 44px tall (links 24px)', small.length === 0, small.join(','));
  await p.screenshot({ path: `${shots}/list-320.png`, fullPage: true });
  await c.close();
}

// 5. Account switching and back/forward cache never resurrect another account's list.
{
  const c = await ctxFor('alice');
  const p = await c.newPage();
  await p.goto('/favorites');
  check('switching: alice sees her partners', (await p.content()).includes('Nick dave'));
  await c.clearCookies();
  await c.addCookies([{ name: 'sid', value: 'bob', domain: host, path: '/' }]);
  await p.goto('/favorites');
  check('switching: bob sees none of alice\'s', !(await p.content()).includes('Nick dave'));
  await p.goBack();
  await p.waitForLoadState();
  check('switching: back button re-fetches (no-store) and does not show alice\'s list to bob', !(await p.content()).includes('Nick dave'));
  await c.clearCookies();
  await c.addCookies([{ name: 'sid', value: 'alice-n2', domain: host, path: '/' }]);
  await p.goto('/favorites');
  check('switching: network 2 session sees no network 1 favorites', !(await p.content()).includes('Nick dave'));
  await c.close();
}

// 6. Dark mode and forced colors render (contrast is covered by axe in light mode only).
{
  const c = await ctxFor('alice', { colorScheme: 'dark' });
  const p = await c.newPage();
  await p.goto('/favorites');
  await axeRun(p, 'list (dark)');
  await p.screenshot({ path: `${shots}/list-dark.png`, fullPage: true });
  await c.close();
}

await browser.close();
fs.writeFileSync(`${shots}/results.json`, JSON.stringify(results, null, 2));
console.log(`${results.length - failed}/${results.length} checks passed`);
process.exit(failed ? 1 : 0);
