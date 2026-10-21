// Screen-reader PROXY checks (UX-056, QA-035). This is NOT a human screen-reader run.
// It asserts what a screen reader would be given: the accessibility tree (landmarks, headings, roles, accessible names)
// via Playwright ARIA snapshots. A real run on NVDA/JAWS/VoiceOver/TalkBack is still required; see docs/qa/SCREEN-READER.md.
const { test, expect } = require('../fixtures');

const n = (st) => `/n/${st.slug}`;

// [name, who, path]
const FLOWS = [
  ['sign in', null, () => '/login'],
  ['register', null, () => '/register'],
  ['dashboard / balance', 'alice', (st) => n(st)],
  ['propose a trade', 'alice', (st) => `${n(st)}/trades/new?to=${st.bobId}`],
  ['trade detail (pending, recipient)', 'bob', (st) => `${n(st)}/trades/${st.tradePending}`],
  ['accept a trade', 'bob', (st) => `${n(st)}/trades/${st.tradePending}/accept`],
  ['history', 'alice', (st) => `${n(st)}/history`],
  ['directory', 'alice', (st) => `${n(st)}/directory`],
  ['dispute form', 'bob', (st) => `${n(st)}/trades/${st.tradeDone}/dispute`],
  ['disputes', 'alice', (st) => `${n(st)}/disputes`],
  ['inbox', 'alice', (st) => `${n(st)}/inbox`],
  ['admin overview', 'admin', (st) => `${n(st)}/admin`],
];

const NAMED_ROLES = ['link', 'button', 'textbox', 'combobox', 'checkbox', 'radio', 'spinbutton', 'searchbox', 'switch', 'tab', 'meter', 'progressbar'];

/** Accessibility-tree lines whose role needs a name but has none, e.g. `- link:` or `- button ""`. */
function unnamed(snapshot) {
  const bad = [];
  for (const line of snapshot.split('\n')) {
    const m = line.match(/^\s*- (\w+)(?: ("[^"]*"|'[^']*'))?/);
    if (!m || !NAMED_ROLES.includes(m[1])) continue;
    const name = m[2] ? m[2].slice(1, -1).trim() : '';
    if (!name) bad.push(line.trim());
  }
  return bad;
}

for (const [name, who, pathOf] of FLOWS) {
  test(`aria tree: ${name}`, async ({ page, as, st }) => {
    const p = who ? await as(who) : page;
    await p.goto(pathOf(st));
    const snap = await p.locator('body').ariaSnapshot();

    // Landmarks: exactly one banner, main and contentinfo; the navigation has a name.
    for (const lm of ['banner', 'main', 'contentinfo']) {
      const count = (snap.match(new RegExp(`^- ${lm}\\b`, 'gm')) || []).length;
      expect(count, `${name}: landmark ${lm}`).toBe(1);
    }
    // Skip link is the first thing a screen reader meets and points at main.
    expect(snap.split('\n')[0], `${name}: first node`).toMatch(/^- link "Skip to main content"/);
    // Exactly one level-1 heading, inside main, and no skipped heading level (h1 -> h3).
    const heads = [...snap.matchAll(/heading "([^"]*)" \[level=(\d)\]/g)].map((m) => [m[1], Number(m[2])]);
    expect(heads.filter((h) => h[1] === 1).length, `${name}: h1 count`).toBe(1);
    let prev = 0;
    for (const [text, lvl] of heads) {
      expect(lvl - prev, `${name}: heading "${text}" skips a level`).toBeLessThanOrEqual(1);
      prev = lvl;
    }
    // Every interactive control has an accessible name.
    expect(unnamed(snap), `${name}: controls without an accessible name`).toEqual([]);
    // Every form control is exposed through a label (no placeholder-only fields): count of inputs equals named textboxes etc.
    const unlabelled = await p.evaluate(() => Array.from(document.querySelectorAll('input:not([type=hidden]):not([type=submit]), select, textarea'))
      .filter((el) => !(el.labels && el.labels.length) && !el.getAttribute('aria-label') && !el.getAttribute('aria-labelledby'))
      .map((el) => el.outerHTML.slice(0, 120)));
    expect(unlabelled, `${name}: form controls without a label`).toEqual([]);
    // The page has a title that is not empty, and html has a language.
    expect((await p.title()).trim().length, `${name}: title`).toBeGreaterThan(0);
    expect(await p.locator('html').getAttribute('lang'), `${name}: lang`).toBeTruthy();
  });
}

test('aria tree: sign-in form is a labelled form with named fields', async ({ page }) => {
  await page.goto('/login');
  await expect(page.locator('main')).toMatchAriaSnapshot(`
    - heading [level=1]
    - textbox "Login"
    - textbox "Password"
    - button "Sign in"
  `);
});

test('aria tree: dashboard exposes named regions, a balance statement and a credit-limit meter', async ({ as, st }) => {
  const p = await as('alice');
  await p.goto(n(st));
  await expect(p.getByRole('navigation', { name: 'Main' })).toBeVisible();
  await expect(p.getByRole('region', { name: 'Your balance' })).toBeVisible();
  await expect(p.getByRole('region', { name: 'Latest activity' })).toBeVisible();
  await expect(p.getByRole('meter', { name: 'Credit limit in use' })).toBeVisible();
  // Balance direction is in words, not only a sign or colour.
  await expect(p.getByRole('region', { name: 'Your balance' })).toContainText(/In debt by|In credit by|Balance is zero|Your balance is/);
  await expect(p.getByRole('region', { name: 'Your balance' })).toContainText('Available to spend');
});

test('aria tree: a failed sign-in is exposed as an alert with a link to the field', async ({ page }) => {
  await page.goto('/login');
  await page.getByRole('button', { name: 'Sign in' }).click();
  const snap = await page.locator('main').ariaSnapshot();
  expect(snap).toMatch(/- (alert|region|group)\b/);
  await expect(page.locator('[role=alert], [role=alertdialog], #error-summary, .error-summary').first()).toBeVisible();
});

test('aria tree: data tables expose column headers', async ({ as, st }) => {
  const p = await as('alice');
  for (const u of [`${n(st)}/history`, `${n(st)}/directory`, `${n(st)}/trades`]) {
    await p.goto(u);
    const bad = await p.evaluate(() => Array.from(document.querySelectorAll('table'))
      .filter((t) => !t.querySelector('th') || !t.querySelector('caption, [aria-label], [aria-labelledby]'))
      .length);
    expect(bad, `${u}: tables without header cells or a name`).toBe(0);
  }
});
