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

  ['admin overview', 'admin', (st) => `${n(st)}/admin`],
  ['admin members', 'admin', (st) => `${n(st)}/admin/members`],
  ['admin member action', 'admin', (st) => `${n(st)}/admin/members/${st.bobId}/suspend`],
  ['admin invitations', 'admin', (st) => `${n(st)}/admin/invitations`],
  ['admin policy', 'admin', (st) => `${n(st)}/admin/policy`],
  ['admin adjustment', 'admin', (st) => `${n(st)}/admin/adjust`],
  ['admin audit log', 'admin', (st) => `${n(st)}/admin/audit`],
  ['admin exports', 'admin', (st) => `${n(st)}/admin/exports`],
  ['review queue', 'admin', (st) => `${n(st)}/review`],
  ['dispute detail (reviewer)', 'admin', (st) => st.disputePath],
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
