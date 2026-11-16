// Automated accessibility check (axe-core, the WCAG 2.2 AA rule set, light and dark colour schemes) of the pages of
// the real binary. Every page template must be visited by at least one check (UX-054, QA-033, ARC-017); see the
// template coverage test at the end.
const fs = require('fs');
const path = require('path');
const { test, expect, expectNoAxeViolations, axeViolations } = require('../fixtures');
const { PAGES, n } = require('../pages');

for (const [name, who, pathOf] of PAGES) {
  test(`axe wcag 2.2 AA, light and dark: ${name}`, async ({ st, as, page: anon }) => {
    const page = who ? await as(who) : anon;
    const resp = await page.goto(pathOf(st));
    expect(resp, 'navigation response').not.toBeNull();
    // Error pages are checked too; only a server fault is a test failure.
    expect(resp.status(), `${name} status`).toBeLessThan(500);
    await expect(page.locator('h1')).toHaveCount(1);
    await expectNoAxeViolations(page, name);
  });
}

test('axe: validation error state of the sign-in form', async ({ page, st }) => {
  await page.goto('/login');
  await page.getByLabel('Login', { exact: true }).fill(st.users.alice);
  await page.getByLabel('Password', { exact: true }).fill('wrong wrong wrong');
  await page.getByRole('button', { name: 'Sign in', exact: true }).click();
  await expect(page.locator('#error-summary')).toBeVisible();
  await expectNoAxeViolations(page, 'login error state');
});

test('axe: validation error state of the propose form', async ({ as, st }) => {
  const page = await as('alice');
  await page.goto(`${'/n/' + st.slug}/trades/new`);
  await page.getByLabel('Amount').fill('not a number');
  await page.getByRole('button', { name: 'Review trade' }).click();
  await expect(page.locator('#error-summary')).toBeVisible();
  await expectNoAxeViolations(page, 'propose error state');
});

test('axe: review step of a proposal', async ({ as, st }) => {
  const page = await as('alice');
  await page.goto(`/n/${st.slug}/trades/new?to=${st.bobId}`);
  await page.getByLabel('Amount').fill('1.00');
  await page.getByRole('button', { name: 'Review trade' }).click();
  await expect(page.getByRole('button', { name: 'Send trade' })).toBeVisible();
  await expectNoAxeViolations(page, 'propose review step');
});

test('axe: time zone validation error', async ({ as }) => {
  const page = await as('alice');
  await page.goto('/account/time-zone');
  await page.getByLabel('Time zone', { exact: true }).fill('Mars/Olympus');
  await page.getByRole('button', { name: 'Save time zone' }).click();
  await expect(page.locator('#error-summary')).toBeVisible();
  await expectNoAxeViolations(page, 'time zone error state');
});

test('axe: time zone saved', async ({ as, st }) => {
  const page = await as('alice');
  await page.goto('/account/time-zone');
  await page.getByLabel('Time zone', { exact: true }).fill('Europe/Berlin');
  await page.getByRole('button', { name: 'Save time zone' }).click();
  await expect(page.locator('h1')).toHaveCount(1);
  await expectNoAxeViolations(page, 'time zone saved');
  await page.goto('/account/time-zone');
  await page.getByLabel('Time zone', { exact: true }).fill('');
  await page.getByRole('button', { name: 'Save time zone' }).click();
});

test('axe: announcement form error and reversal review and error states', async ({ as, st }) => {
  const page = await as('admin');
  await page.goto(`/n/${st.slug}/admin/announcements`);
  await page.getByRole('button', { name: 'Publish announcement' }).click();
  await expect(page.locator('#error-summary')).toBeVisible();
  await expectNoAxeViolations(page, 'announcement error state');

  await page.goto(`/n/${st.slug}/admin/reverse`);
  await page.getByRole('button', { name: 'Review reversal' }).click();
  await expect(page.locator('#error-summary')).toBeVisible();
  await expectNoAxeViolations(page, 'reverse error state');

  await page.goto(`/n/${st.slug}/admin/reverse`);
  await page.getByLabel('Trade reference').fill(st.tradeReverse);
  await page.getByLabel('Reason').fill('Goods were never handed over.');
  await page.getByRole('button', { name: 'Review reversal' }).click();
  await expect(page.getByRole('button', { name: 'Post reversal' })).toBeVisible();
  await expectNoAxeViolations(page, 'reverse review step');
});

test('axe: dispute decision review step and empty response error', async ({ as, st }) => {
  const carol = await as('carol');
  await carol.goto(st.disputePath);
  await carol.getByRole('button', { name: 'Send response' }).click();
  await expect(carol.locator('#error-summary')).toBeVisible();
  await expectNoAxeViolations(carol, 'dispute response error state');

  const admin = await as('admin');
  await admin.goto(st.disputePath);
  await admin.getByRole('button', { name: 'Start review' }).click();
  await admin.getByLabel('Rationale').fill('Looked at both sides.');
  await admin.getByRole('button', { name: 'Review decision' }).click();
  await expect(admin.getByRole('button', { name: 'Record decision' })).toBeVisible();
  await expectNoAxeViolations(admin, 'dispute decision review step');
});

// Guard against a false green: the harness must report a violation when one exists.
test('axe harness detects a deliberately broken page', async ({ page }) => {
  await page.goto('/login');
  await page.evaluate(() => {
    document.querySelector('main').insertAdjacentHTML('beforeend', '<img src="/static/none.png"><input type="text">');
  });
  const ids = (await axeViolations(page)).map((v) => v.id);
  expect(ids).toContain('image-alt');
  expect(ids).toContain('label');
});

// The same guard for the WCAG 2.2 tag and the dark pass: crowded tiny buttons (2.5.8, wcag22aa) are reported, and a
// contrast failure that exists only in the dark scheme fails the check, named with its scheme.
test('axe harness detects a target-size violation and a dark-only contrast failure', async ({ page }) => {
  await page.goto('/login');
  await page.addStyleTag({ content: '.tiny{min-height:0;min-width:0;width:10px;height:10px;padding:0;margin:0;border:0}' +
    '@media (prefers-color-scheme: dark){main p{color:#2a2a2a!important}}' });
  await page.evaluate(() => {
    const box = document.createElement('div');
    for (const name of ['first', 'second']) {
      const b = document.createElement('button');
      b.className = 'tiny';
      b.setAttribute('aria-label', name);
      box.appendChild(b);
    }
    document.querySelector('main').appendChild(box);
  });
  expect((await axeViolations(page)).map((v) => v.id)).toContain('target-size');
  await page.locator('.tiny').evaluateAll((els) => els.forEach((e) => e.remove()));
  expect(await axeViolations(page), 'light scheme is clean').toEqual([]);
  const err = await expectNoAxeViolations(page, 'dark-only guard').catch((e) => e);
  expect(err && err.message).toMatch(/dark-only guard \(dark\): \d+ axe violation\(s\)[\s\S]*color-contrast/);
  expect(err.message).not.toContain('dark-only guard (light)');
});

// Maintenance mode (OPS-024): reads still work, a write gets the 503 "Back soon" page. A second instance with the
// maintenance setting on is started so the shared one is not disturbed.
test('axe: maintenance page, and reads keep working in maintenance mode', async ({ page }) => {
  const { startInstance } = require('../global-setup');
  const inst = await startInstance({ CIRCULA_MAINTENANCE: 'true' });
  try {
    const read = await page.request.get(`${inst.base}/login`);
    expect(read.status(), 'GET in maintenance').toBe(200);
    const write = await page.request.post(`${inst.base}/login`, { form: { login: 'x', password: 'y' } });
    expect(write.status(), 'POST in maintenance').toBe(503);
    expect(write.headers()['retry-after']).toBeTruthy();
    await page.setContent(await write.text());
    await expect(page.locator('h1')).toHaveText('Back soon');
    await expectNoAxeViolations(page, 'maintenance page');
  } finally {
    await inst.stop();
  }
});

// UX-061: balance at an exact instant on the History page, keyboard only.
test('axe and keyboard: historical balance lookup, error then result', async ({ as, st }) => {
  const { tabTo } = require('../kbd');
  const page = await as('alice');
  await page.goto(`${n(st)}/history`);
  await tabTo(page, /Date and time/);
  await page.keyboard.type('2026-03-01');
  await Promise.all([page.waitForNavigation(), page.keyboard.press('Enter')]);
  await expect(page.locator('#histbal-err')).toContainText('Date and time: use a full RFC 3339 date-time');
  await expect(page.locator('#histbal-at')).toHaveValue('2026-03-01');
  await expect(page.locator('#histbal-at')).toHaveAttribute('aria-invalid', 'true');
  await expectNoAxeViolations(page, 'historical balance field error');

  const at = new Date(Date.now() - 1000).toISOString(); // Z, millisecond precision, not in the future
  await page.locator('#histbal-at').fill('');
  await tabTo(page, /Date and time/);
  await page.keyboard.type(at);
  await Promise.all([page.waitForNavigation(), page.keyboard.press('Enter')]);
  await expect(page.locator('#histbal-result')).toContainText('posted balance of your account at this instant');
  await expect(page.locator('#histbal-result time')).toHaveAttribute('datetime', at);
  await expectNoAxeViolations(page, 'historical balance result');
});

// Template coverage (UX-054, QA-033, ARC-017: checks run on every page template). Each template in
// internal/web/templates, other than the base layout, must be rendered by at least one axe check: a PAGES entry (pages.js)
// or a labelled check in a spec file listed here. A new template fails this test until such a check is added.
const TEMPLATES_DIR = path.join(__dirname, '..', '..', 'internal', 'web', 'templates');
const LAYOUT_ONLY = ['base'];
// template -> [spec file, label passed to expectNoAxeViolations or checkPage on a page that renders it]
const STATE_CHECKS = {
  propose_review: ['a11y.spec.js', 'propose review step'],
  admin_reverse_review: ['a11y.spec.js', 'reverse review step'],
  dispute_decide_review: ['a11y.spec.js', 'dispute decision review step'],
  admin_check: ['a11y-governance.spec.js', 'consistency check result'],
  admin_adjust_review: ['a11y-governance.spec.js', 'adjustment review step'],
  pending: ['a11y-governance.spec.js', 'waiting for approval'],
  networks: ['a11y-governance.spec.js', 'networks list (no network yet)'],
  security: ['security.spec.js', 'security (two-step off)'],
  mfa_enrol: ['security.spec.js', 'two-step set-up start'],
  mfa_recovery: ['security.spec.js', 'two-step recovery codes shown'],
  login_mfa: ['security.spec.js', 'two-step challenge'],
  recovery_codes: ['security.spec.js', 'recovery codes (none yet)'],
  recover: ['security.spec.js', 'recover form'],
  admin_reset_done: ['security.spec.js', 'temporary password shown'],
  admin_break_glass: ['security.spec.js', 'operator access'],
  admin_adjust_pending: ['security.spec.js', 'pending adjustments (empty)'],
  erase_confirm: ['security.spec.js', 'erase my data'],
};

function axeLabels(file) {
  const src = fs.readFileSync(path.join(__dirname, file), 'utf8');
  return new Set([...src.matchAll(/(?:expectNoAxeViolations|checkPage)\([\w.]+, '([^']+)'\)/g)].map((m) => m[1]));
}

test('every page template is checked by axe', () => {
  const templates = fs.readdirSync(TEMPLATES_DIR).filter((f) => f.endsWith('.html')).map((f) => f.slice(0, -5));
  expect(templates.length, 'templates found').toBeGreaterThan(50);
  const known = new Set(templates);
  const covered = new Set();
  for (const [name, , , tpl] of PAGES) {
    expect(known.has(tpl), `PAGES "${name}" names template ${tpl}, which does not exist`).toBe(true);
    covered.add(tpl);
  }
  for (const [tpl, [file, label]] of Object.entries(STATE_CHECKS)) {
    expect(known.has(tpl), `STATE_CHECKS names template ${tpl}, which does not exist`).toBe(true);
    expect(axeLabels(file).has(label), `${file} has no axe check labelled "${label}" (for ${tpl})`).toBe(true);
    covered.add(tpl);
  }
  const missing = templates.filter((t) => !LAYOUT_ONLY.includes(t) && !covered.has(t));
  expect(missing, 'templates no axe check renders').toEqual([]);
});
