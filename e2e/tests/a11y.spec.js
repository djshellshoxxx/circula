// Automated accessibility check (axe-core, WCAG 2.x A and AA) of the main pages of the real binary.
const { test, expect, expectNoAxeViolations, axeViolations } = require('../fixtures');

const n = (st) => `/n/${st.slug}`;

// [name, who (null = anonymous), path builder]
const PAGES = [
  ['login', null, () => '/login'],
  ['register', null, () => '/register'],
  ['help (anonymous)', null, () => '/help'],
  ['invitation, signed out', null, (st) => st.unusedInvite],
  ['not found', null, () => '/no-such-page'],

  ['networks list', 'carol', () => '/'],
  ['setup (network creation)', 'carol', () => '/setup'],
  ['password change', 'alice', () => '/account/password'],
  ['help (member)', 'alice', () => '/help'],
  ['invitation, signed in', 'alice', (st) => st.unusedInvite],

  ['dashboard', 'alice', (st) => n(st)],
  ['dashboard with pending trade', 'bob', (st) => n(st)],
  ['directory', 'alice', (st) => `${n(st)}/directory`],
  ['member profile', 'alice', (st) => st.bobProfile],
  ['listings', 'bob', (st) => `${n(st)}/listings`],
  ['new listing', 'alice', (st) => `${n(st)}/listings/new`],
  ['edit listing', 'alice', (st) => st.listingEdit],
  ['trades', 'alice', (st) => `${n(st)}/trades`],
  ['propose a trade', 'alice', (st) => `${n(st)}/trades/new?to=${st.bobId}`],
  ['trade detail (completed)', 'alice', (st) => `${n(st)}/trades/${st.tradeDone}`],
  ['trade detail (pending, recipient)', 'bob', (st) => `${n(st)}/trades/${st.tradePending}`],
  ['accept a trade', 'bob', (st) => `${n(st)}/trades/${st.tradePending}/accept`],
  ['decline a trade', 'bob', (st) => `${n(st)}/trades/${st.tradePending}/decline`],
  ['cancel a trade', 'carol', (st) => `${n(st)}/trades/${st.tradePending}/cancel`],
  ['history', 'alice', (st) => `${n(st)}/history`],
  ['dispute form', 'bob', (st) => `${n(st)}/trades/${st.tradeDone}/dispute`],
  ['disputes', 'alice', (st) => `${n(st)}/disputes`],
  ['dispute detail', 'alice', (st) => st.disputePath],
  ['dispute detail (other party, response form)', 'carol', (st) => st.disputePath],
  ['dispute detail (resolved, with response, re-review form)', 'alice', (st) => st.disputeResolved],
  ['dispute detail (resolved, responder view)', 'bob', (st) => st.disputeResolved],
  ['dispute detail (re-review requested, requester)', 'alice', (st) => st.disputeReReview],
  ['dispute detail (re-review decision form)', 'carol', (st) => st.disputeReReview],
  ['inbox (with notifications)', 'alice', (st) => `${n(st)}/inbox`],
  ['inbox (waiting for action)', 'bob', (st) => `${n(st)}/inbox`],
  ['inbox (unread only)', 'carol', (st) => `${n(st)}/inbox?show=unread`],
  ['dashboard with announcement', 'carol', (st) => n(st)],
  ['report a listing', 'carol', (st) => st.reportPath],
  ['report a member', 'alice', (st) => `${n(st)}/report?type=member&id=${st.bobId}`],
  ['time zone', 'alice', () => '/account/time-zone'],

  ['admin overview', 'admin', (st) => `${n(st)}/admin`],
  ['admin members', 'admin', (st) => `${n(st)}/admin/members`],
  ['admin member action', 'admin', (st) => `${n(st)}/admin/members/${st.bobId}/suspend`],
  ['admin invitations', 'admin', (st) => `${n(st)}/admin/invitations`],
  ['admin policy', 'admin', (st) => `${n(st)}/admin/policy`],
  ['admin adjustment', 'admin', (st) => `${n(st)}/admin/adjust`],
  ['admin audit log', 'admin', (st) => `${n(st)}/admin/audit`],
  ['admin exports', 'admin', (st) => `${n(st)}/admin/exports`],
  ['admin reports (open)', 'admin', (st) => `${n(st)}/admin/reports`],
  ['admin reports (all)', 'admin', (st) => `${n(st)}/admin/reports?status=all`],
  ['admin announcements', 'admin', (st) => `${n(st)}/admin/announcements`],
  ['admin reverse a trade', 'admin', (st) => `${n(st)}/admin/reverse`],
  ['admin reverse a trade (trade chosen)', 'admin', (st) => `${n(st)}/admin/reverse?tx=${st.tradeReverse}`],
  ['admin overview (needs attention)', 'admin', (st) => `${n(st)}/admin`],
  ['review queue', 'admin', (st) => `${n(st)}/review`],
  ['dispute detail (reviewer)', 'admin', (st) => st.disputePath],
  ['dispute detail (re-review waiting, first reviewer)', 'admin', (st) => st.disputeReReview],
];

for (const [name, who, pathOf] of PAGES) {
  test(`axe wcag2a/2aa: ${name}`, async ({ st, as, page: anon }) => {
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
