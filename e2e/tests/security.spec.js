// Accessibility (axe) and keyboard-only coverage of the security pages: Security, two-step sign-in (set-up and the
// sign-in challenge), recovery codes and account recovery, administrator password reset, operator access
// (break-glass), erasure, dual-control adjustments, password change, and the read-only page of a frozen network.
//
// They run against their own instance (see secfixture.js) so that changing passwords, turning on two-step sign-in
// or erasing a member does not disturb the shared seed. The tests are order-dependent and run serially.
const { test, expect } = require('@playwright/test');
const { buildSecurityInstance, totp, PASSWORD, SLUG } = require('../secfixture');
const { tabTo, kbSubmit, kbActivate } = require('../kbd');

const AXE_PATH = require.resolve('axe-core/axe.min.js');
const AXE_TAGS = ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'];
const NEW_PASSWORD = 'a different long passphrase 42';

test.describe.configure({ mode: 'serial' });
test.setTimeout(120_000);

let sec; // the instance
const state = {}; // ids and secrets passed between the ordered tests

test.beforeAll(async () => {
  test.setTimeout(240_000);
  sec = await buildSecurityInstance();
});
test.afterAll(async () => {
  if (sec) await sec.cleanup();
});

async function newPage(browser) {
  const ctx = await browser.newContext({ baseURL: sec.base, bypassCSP: true });
  return { ctx, page: await ctx.newPage() };
}

async function expectClean(page, label) {
  await page.addScriptTag({ path: AXE_PATH });
  const v = await page.evaluate(async (tags) => {
    // eslint-disable-next-line no-undef
    const r = await axe.run(document, { runOnly: { type: 'tag', values: tags }, resultTypes: ['violations'] });
    return r.violations.map((x) => ({ id: x.id, impact: x.impact, help: x.help, nodes: x.nodes.map((n) => ({ target: n.target.join(' '), html: n.html.slice(0, 200) })) }));
  }, AXE_TAGS);
  if (v.length) {
    throw new Error(`${label}: ${v.length} axe violation(s) at ${page.url()}\n` +
      v.map((x) => `  [${x.impact}] ${x.id}: ${x.help}\n` + x.nodes.map((n) => `      ${n.target}\n      ${n.html}`).join('\n')).join('\n'));
  }
}

async function checkPage(page, label) {
  await expect(page.locator('h1'), `${label}: one h1`).toHaveCount(1);
  await expectClean(page, label);
}

async function signIn(page, who, password = PASSWORD) {
  await page.goto('/login');
  await page.getByLabel('Login', { exact: true }).fill(sec.users[who]);
  await page.getByLabel('Password', { exact: true }).fill(password);
  await Promise.all([page.waitForLoadState('load'), page.getByRole('button', { name: 'Sign in', exact: true }).click()]);
}

async function memberId(page, display) {
  await page.goto(`/n/${SLUG}/directory`);
  const href = await page.locator(`a[href*="/n/${SLUG}/members/"]`).filter({ hasText: display }).first().getAttribute('href');
  return href.split('/').pop();
}

// ---- two-step sign-in ----

test('axe: security page, two-step set-up and recovery codes shown once', async ({ browser }) => {
  const { ctx, page } = await newPage(browser);
  await signIn(page, 'alice');
  await page.goto('/account/security');
  await checkPage(page, 'security (two-step off)');
  await page.goto('/account/security/mfa/enrol');
  await checkPage(page, 'two-step set-up start');
  await page.getByLabel('Password').fill(PASSWORD);
  await Promise.all([page.waitForLoadState('load'), page.getByRole('button', { name: 'Start setup' }).click()]);
  await expect(page.getByLabel('Code from your app')).toBeVisible();
  await checkPage(page, 'two-step set-up with key');
  state.secret = (await page.locator('main code').first().innerText()).trim();

  // wrong code: error state
  await page.getByLabel('Code from your app').fill('000000');
  await page.getByLabel('Password').fill(PASSWORD);
  await Promise.all([page.waitForLoadState('load'), page.getByRole('button', { name: 'Turn on two-step sign-in' }).click()]);
  await expect(page.locator('#error-summary')).toBeVisible();
  await checkPage(page, 'two-step set-up wrong code');

  await page.getByLabel('Code from your app').fill(totp(state.secret));
  await page.getByLabel('Password').fill(PASSWORD);
  await Promise.all([page.waitForLoadState('load'), page.getByRole('button', { name: 'Turn on two-step sign-in' }).click()]);
  await expect(page.getByText('Save these codes now')).toBeVisible();
  await checkPage(page, 'two-step recovery codes shown');
  state.mfaCode = (await page.locator('main ul.rows code').first().innerText()).trim();

  await page.goto('/account/security');
  await expect(page.getByText('is on')).toBeVisible();
  await checkPage(page, 'security (two-step on)');
  await ctx.close();
});

test('axe and keyboard: two-step sign-in challenge', async ({ browser }) => {
  const { ctx, page } = await newPage(browser);
  await page.goto('/login');
  await page.getByLabel('Login', { exact: true }).fill(sec.users.alice);
  await page.getByLabel('Password', { exact: true }).fill(PASSWORD);
  await Promise.all([page.waitForLoadState('load'), page.getByRole('button', { name: 'Sign in', exact: true }).click()]);
  await expect(page.getByLabel('Authentication or recovery code')).toBeVisible();
  await checkPage(page, 'two-step challenge');

  // wrong code: error state, still accessible
  await page.getByLabel('Authentication or recovery code').fill('000000');
  await Promise.all([page.waitForLoadState('load'), page.getByRole('button', { name: 'Continue' }).click()]);
  await expect(page.locator('#error-summary')).toBeVisible();
  await checkPage(page, 'two-step challenge wrong code');

  // Keyboard only: use a saved recovery code (the authenticator step was used up by the set-up).
  await kbSubmit(page, [[/^Authentication or recovery code$/, state.mfaCode]], /^Continue$/);
  await expect(page).not.toHaveURL(/\/login/);
  await expect(page.locator('#error-summary')).toHaveCount(0);
  await ctx.close();
});

// ---- recovery codes and account recovery ----

test('keyboard and axe: make recovery codes, then recover the account without email', async ({ browser }) => {
  const { ctx, page } = await newPage(browser);
  await signIn(page, 'bob');
  await page.goto('/account/recovery-codes');
  await checkPage(page, 'recovery codes (none yet)');
  await page.getByRole('button', { name: 'Make new recovery codes' }).click(); // empty password: error state
  await expect(page.locator('#error-summary')).toBeVisible();
  await checkPage(page, 'recovery codes error state');
  await page.goto('/account/recovery-codes');
  await kbSubmit(page, [[/^Current password$/, PASSWORD]], /^Make new recovery codes$/);
  await expect(page.getByText('Your new recovery codes')).toBeVisible();
  await checkPage(page, 'recovery codes shown');
  state.recoveryCode = (await page.locator('ol.codes code').first().innerText()).trim();
  await ctx.close();

  const anon = await newPage(browser);
  await anon.page.goto('/recover');
  await checkPage(anon.page, 'recover form');
  await anon.page.getByRole('button', { name: 'Set new password' }).click(); // empty: error state
  await expect(anon.page.locator('#error-summary')).toBeVisible();
  await checkPage(anon.page, 'recover form error state');
  await anon.page.goto('/recover');
  await kbSubmit(anon.page, [
    [/^Login$/, sec.users.bob], [/^Recovery code$/, state.recoveryCode],
    [/^New password$/, NEW_PASSWORD], [/^Confirm new password$/, NEW_PASSWORD],
  ], /^Set new password$/);
  await expect(anon.page).toHaveURL(/\/login/);
  await expect(anon.page.locator('#flash')).toContainText('Your password was changed');
  await checkPage(anon.page, 'sign-in after recovery');
  await signIn(anon.page, 'bob', NEW_PASSWORD);
  await expect(anon.page).not.toHaveURL(/\/login/);
  await anon.ctx.close();
});

test('keyboard and axe: administrator resets a password', async ({ browser }) => {
  const { ctx, page } = await newPage(browser);
  await signIn(page, 'admin');
  state.bobId = await memberId(page, 'Bob');
  await page.goto(`/n/${SLUG}/admin/members/${state.bobId}/reset-password`);
  await checkPage(page, 'reset password confirmation');
  await page.getByRole('button', { name: /Reset password/ }).click(); // no reason: error state
  await expect(page.locator('#error-summary')).toBeVisible();
  await checkPage(page, 'reset password error state');
  await page.goto(`/n/${SLUG}/admin/members/${state.bobId}/reset-password`);
  await kbSubmit(page, [[/^Reason$/, 'Lost the recovery codes']], /^Reset password/);
  await expect(page.locator('code.temp')).toBeVisible();
  await checkPage(page, 'temporary password shown');
  await ctx.close();
});

// ---- operator access (break-glass) ----

test('axe and keyboard: operator access page lists the session', async ({ browser }) => {
  const { ctx, page } = await newPage(browser);
  await signIn(page, 'admin');
  await page.goto(`/n/${SLUG}/admin`);
  await kbActivate(page, /^Operator access/);
  await expect(page).toHaveURL(/\/admin\/break-glass$/);
  await expect(page.getByText('ops-e2e')).toBeVisible();
  await checkPage(page, 'operator access');
  // the scrollable table region is reachable by keyboard
  await tabTo(page, /^Operator access table$/);
  await ctx.close();
});

// ---- dual control ----

test('axe and keyboard: dual-control pending adjustments, approve with a second administrator', async ({ browser }) => {
  const a = await newPage(browser);
  await signIn(a.page, 'admin');
  await a.page.goto(`/n/${SLUG}/admin/adjustments/pending`);
  await expect(a.page.getByText('No requests match this filter')).toBeVisible();
  await checkPage(a.page, 'pending adjustments (empty)');

  const request = async (amount, reason) => {
    await a.page.goto(`/n/${SLUG}/admin/adjust`);
    await a.page.getByLabel('Member', { exact: true }).selectOption({ label: 'Alice Member' });
    await a.page.getByLabel('Amount').fill(amount);
    await a.page.getByLabel('Reason').fill(reason);
    await Promise.all([a.page.waitForLoadState('load'), a.page.getByRole('button', { name: 'Review adjustment' }).click()]);
    await Promise.all([a.page.waitForLoadState('load'), a.page.getByRole('button', { name: 'Post adjustment' }).click()]);
  };
  await request('5.00', 'Goods credit one');
  await expect(a.page.locator('#flash')).toContainText('another administrator must approve');
  await request('6.00', 'Goods credit two');

  await a.page.goto(`/n/${SLUG}/admin/adjustments/pending`);
  await expect(a.page.getByRole('button', { name: /Cancel request/ }).first()).toBeVisible();
  await expect(a.page.getByRole('button', { name: /Approve and post/ })).toHaveCount(0); // not your own
  await checkPage(a.page, 'pending adjustments (proposer view)');

  const b = await newPage(browser);
  await signIn(b.page, 'admin2');
  await b.page.goto(`/n/${SLUG}/admin/adjustments/pending`);
  await expect(b.page.getByRole('button', { name: /Approve and post/ }).first()).toBeVisible();
  await checkPage(b.page, 'pending adjustments (second administrator)');
  // keyboard only: Tab to the first Approve button and press Enter
  await tabTo(b.page, /^Approve and post/);
  await Promise.all([b.page.waitForNavigation(), b.page.keyboard.press('Enter')]);
  await expect(b.page.locator('#flash')).toBeVisible();
  await checkPage(b.page, 'pending adjustments after approval');
  await b.page.goto(`/n/${SLUG}/admin/adjustments/pending?status=approved`);
  await checkPage(b.page, 'approved adjustments');
  await a.ctx.close();
  await b.ctx.close();
});

// ---- erasure ----

test('axe and keyboard: erase my data and administrator erasure', async ({ browser }) => {
  const a = await newPage(browser);
  await signIn(a.page, 'admin');
  const dora = await memberId(a.page, 'Dora');
  const eve = await memberId(a.page, 'Eve');
  for (const id of [dora, eve]) {
    await a.page.goto(`/n/${SLUG}/admin/members/${id}/remove`);
    await a.page.getByLabel('Reason').fill('Moved away');
    await Promise.all([a.page.waitForLoadState('load'), a.page.getByRole('button', { name: /^Remove/ }).click()]);
  }

  // Administrator erasure of a departed member
  await a.page.goto(`/n/${SLUG}/admin/members/${eve}/erase`);
  await checkPage(a.page, 'administrator erasure');
  await a.page.getByLabel('Reason').fill('Request of the person');
  await Promise.all([a.page.waitForLoadState('load'), a.page.getByRole('button', { name: 'Erase personal data' }).click()]);
  await expect(a.page.locator('#error-summary')).toBeVisible(); // box not ticked
  await checkPage(a.page, 'administrator erasure error state');
  await a.page.goto(`/n/${SLUG}/admin/members/${eve}/erase`);
  await kbSubmit(a.page, [[/^Reason/, 'Request of the person'], [/I understand/, true]], /^Erase personal data$/);
  await expect(a.page).toHaveURL(/status=departed/);
  await checkPage(a.page, 'departed members after erasure');
  await a.ctx.close();

  // A departed member erases their own data
  const d = await newPage(browser);
  await signIn(d.page, 'dora');
  await d.page.goto(`/n/${SLUG}/erase-my-data`);
  await checkPage(d.page, 'erase my data');
  await Promise.all([d.page.waitForLoadState('load'), d.page.getByRole('button', { name: 'Erase personal data' }).click()]);
  await expect(d.page.locator('#error-summary')).toBeVisible();
  await checkPage(d.page, 'erase my data error state');
  await d.page.goto(`/n/${SLUG}/erase-my-data`);
  await kbSubmit(d.page, [[/I understand/, true]], /^Erase personal data$/);
  await expect(d.page.locator('#flash')).toContainText('has been erased');
  await checkPage(d.page, 'after erase my data');
  await d.ctx.close();
});

// ---- password change ----

test('axe and keyboard: password change', async ({ browser }) => {
  const { ctx, page } = await newPage(browser);
  await signIn(page, 'admin2');
  await page.goto('/account/password');
  await checkPage(page, 'password change');
  await page.getByRole('button', { name: 'Change password' }).click();
  await expect(page.locator('#error-summary')).toBeVisible();
  await checkPage(page, 'password change error state');
  await page.goto('/account/password');
  await kbSubmit(page, [[/^Current password$/, PASSWORD], [/^New password$/, NEW_PASSWORD], [/^Confirm new password$/, NEW_PASSWORD]], /^Change password$/);
  await expect(page).toHaveURL(/\/login/);
  await checkPage(page, 'sign-in after password change');
  await signIn(page, 'admin2', NEW_PASSWORD);
  await expect(page).not.toHaveURL(/\/login/);
  await ctx.close();
});

// ---- frozen network ----

test('frozen network: banner for administrators, notice for members, 423 page, reads keep working', async ({ browser }) => {
  await sec.restartWithBrokenLedger();

  const a = await newPage(browser);
  await signIn(a.page, 'admin');
  await a.page.goto(`/n/${SLUG}/admin`);
  await expect(a.page.getByRole('heading', { name: 'This network is read-only' })).toBeVisible();
  await expect(a.page.getByText(/ledger finding/)).toBeVisible();
  await checkPage(a.page, 'admin overview, frozen');
  await a.ctx.close();

  const c = await newPage(browser);
  await signIn(c.page, 'admin2', NEW_PASSWORD);
  for (const p of [`/n/${SLUG}`, `/n/${SLUG}/listings/new`, `/n/${SLUG}/trades/new`, `/n/${SLUG}/history`]) {
    const r = await c.page.goto(p);
    expect(r.status(), p).toBe(200);
    await expect(c.page.getByText('read-only while its ledger is being checked')).toBeVisible();
    await expect(c.page.getByText(/ledger finding/)).toHaveCount(0); // the reason is for administrators
  }
  await c.page.goto(`/n/${SLUG}/listings/new`);
  await checkPage(c.page, 'write form on a frozen network');

  // A write is refused with the read-only page, status 423.
  await c.page.getByLabel('Title').fill('Should not be saved');
  const [resp] = await Promise.all([
    c.page.waitForResponse((r) => r.request().method() === 'POST' && r.url().includes('/listings')),
    c.page.getByRole('button', { name: 'Publish listing' }).click(),
  ]);
  expect(resp.status()).toBe(423);
  await expect(c.page.getByRole('heading', { name: 'Network read-only' })).toBeVisible();
  await checkPage(c.page, '423 network read-only page');
  await c.ctx.close();
});
