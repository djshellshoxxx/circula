// UX-002: reach core and adjacent existing screens FROM Home in at most
// three rendered-link activations. Measured paths use GET navigation only.
// Rows whose role is "apply:<name>" run in the approval-mode instance (apply.js).
const { test, expect, homeOf } = require('../apply');

const VIEWPORTS = [
  { name: 'portrait', width: 320, height: 640 },
  { name: 'landscape', width: 640, height: 320 },
];
const n = (st) => `/n/${st.slug}`;
const route = (suffix) => (st) => n(st) + suffix;
const admin = route('/admin');
const history = route('/history');
const trades = route('/trades');
const directory = route('/directory');
const listings = route('/listings');
const disputes = route('/disputes');
const review = route('/review');
const pending = (st) => `${n(st)}/trades/${st.tradePending}`;
const fixed = (path) => () => path;
const apply = (suffix) => (st) => `/n/${st.apply.slug}${suffix}`;
const applyRequests = apply('/admin/members?status=pending');

// [screen, role, actual link destinations in order]
const PATHS = [
  ['directory', 'alice', [directory]],
  ['member profile', 'alice', [directory, (st) => st.bobProfile]],
  ['listings', 'alice', [listings]],
  ['new listing', 'alice', [listings, route('/listings/new')]],
  ['edit own listing', 'alice', [listings, (st) => st.listingEdit]],
  ['proposal form', 'alice', [route('/trades/new')]],
  ['trades', 'alice', [trades]],
  ['own historical trade', 'alice', [history, (st) => `${n(st)}/trades/${st.tradeDone}`]],
  ['pending trade', 'bob', [trades, pending]],
  ['acceptance review', 'bob', [trades, pending, (st) => pending(st) + '/accept']],
  ['decline confirmation', 'bob', [trades, pending, (st) => pending(st) + '/decline']],
  ['cancel own proposal', 'carol', [trades, pending, (st) => pending(st) + '/cancel']],
  ['history', 'alice', [history]],
  ['disputes', 'alice', [disputes]],
  ['own dispute detail', 'alice', [disputes, (st) => st.disputePath]],
  ['review queue', 'admin', [review]],
  ['dispute reviewer detail', 'admin', [review, (st) => st.disputePath]],
  ['admin overview', 'admin', [admin]],
  ['admin members', 'admin', [admin, route('/admin/members')]],
  ['admin policy', 'admin', [admin, route('/admin/policy')]],
  ['admin adjustment form', 'admin', [admin, route('/admin/adjust')]],
  ['admin audit and checks', 'admin', [admin, route('/admin/audit')]],
  ['admin invitations', 'admin', [admin, route('/admin/invitations')]],
  ['admin announcements', 'admin', [admin, route('/admin/announcements')]],
  ['admin exports', 'admin', [admin, route('/admin/exports')]],
  ['admin reports', 'admin', [admin, route('/admin/reports')]],
  ['account time zone', 'alice', [() => '/account/time-zone']],
  ['account password form', 'alice', [() => '/account/password']],
  ['Help', 'alice', [() => '/help']],
  ['account security', 'alice', [fixed('/account/security')]],
  ['two-step sign-in setup', 'alice', [fixed('/account/security'), fixed('/account/security/mfa/enrol')]],
  ['password recovery codes', 'alice', [fixed('/account/security'), fixed('/account/recovery-codes')]],
  ['inbox', 'alice', [route('/inbox')]],
  ['inbox, unread only', 'alice', [route('/inbox'), route('/inbox?show=unread')]],
  ['network rules and change log', 'alice', [route('/policy')]],
  ['leave the network (blocked)', 'alice', [route('/policy'), route('/leave')]],
  ['own account activity log', 'alice', [history, route('/audit')]],
  ['admin member action', 'admin', [admin, route('/admin/members'), (st) => `${n(st)}/admin/members/${st.bobId}/suspend`]],
  ['admin inactive members', 'admin', [admin, route('/admin/members'), route('/admin/inactive')]],
  ['admin reverse a trade', 'admin', [admin, route('/admin/reverse')]],
  ['admin adjustments awaiting approval', 'admin', [admin, route('/admin/adjustments/pending')]],
  ['admin pause or close', 'admin', [admin, route('/admin/network')]],
  ['admin operator access', 'admin', [admin, route('/admin/break-glass')]],
  ['admin network terms', 'admin', [admin, route('/admin/terms')]],
  ['network terms (published)', 'apply:hana', [apply('/policy'), apply('/terms')]],
  ['leave the network (form)', 'apply:hana', [apply('/policy'), apply('/leave')]],
  ['admin membership requests', 'apply:erin', [apply('/admin'), applyRequests]],
  ['admin approve a request', 'apply:erin', [apply('/admin'), applyRequests, (st) => `/n/${st.apply.slug}/admin/members/${st.apply.gailId}/approve`]],
];

// Before Home: the sign-in screen leads to the other signed-out entry points.
const SIGNED_OUT = [
  ['registration', [fixed('/register')]],
  ['account recovery', [fixed('/recover')]],
];

async function startAtHome(page, st, home = n(st)) {
  const response = await page.goto(home);
  expect(response).not.toBeNull();
  expect(response.status()).toBe(200);
  await expect(page).toHaveURL((url) => url.pathname.replace(/\/$/, '') === home);
  await expect(page.locator('h1')).toHaveText('Home');
  await expect(page.getByRole('region', { name: 'Your balance', exact: true })).toBeVisible();
}

async function followPath(page, st, destinations) {
  expect(destinations.length, 'rendered link activations from Home').toBeGreaterThan(0);
  expect(destinations.length, 'UX-002 activation budget').toBeLessThanOrEqual(3);
  for (const href of destinations) {
    const link = page.locator(`a[href="${href}"]`).first();
    await expect(link, `${href}: an actual rendered link is available`).toBeVisible();
    const [response] = await Promise.all([page.waitForNavigation(), link.click()]);
    expect(response, `${href}: navigation response`).not.toBeNull();
    expect(response.status(), `${href}: successful destination`).toBe(200);
    const intended = new URL(href, st.base);
    await expect(page).toHaveURL((url) => url.pathname.replace(/\/$/, '') === intended.pathname.replace(/\/$/, '') && url.search === intended.search);
    await expect(page.locator('h1')).toHaveCount(1);
    await expect(page.locator('h1')).toBeVisible();
  }
}

for (const viewport of VIEWPORTS) {
  for (const [name, who, pathOf] of PATHS) {
    test(`UX-002: ${name} from Home within three activations, ${viewport.name}`, async ({ open, st }) => {
      const page = await open(who);
      await page.setViewportSize({ width: viewport.width, height: viewport.height });
      await startAtHome(page, st, homeOf(st, who));
      await followPath(page, st, pathOf.map((path) => path(st)));
    });
  }

  for (const [name, pathOf] of SIGNED_OUT) {
    test(`UX-002: ${name} from sign-in within three activations, ${viewport.name}`, async ({ page, st }) => {
      await page.setViewportSize({ width: viewport.width, height: viewport.height });
      const response = await page.goto('/login');
      expect(response).not.toBeNull();
      expect(response.status()).toBe(200);
      await expect(page.locator('h1')).toHaveText('Sign in');
      await followPath(page, st, pathOf.map((path) => path(st)));
    });
  }

  test(`UX-002: dispute raise from Home in three activations, ${viewport.name}`, async ({ as, st }) => {
    // Prepare only our own completed synthetic trade. The seed's completed trades
    // already have disputes/reversals; do not repurpose or mutate their IDs.
    const alice = await as('alice');
    const bob = await as('bob');
    await alice.goto(`${n(st)}/trades/new?to=${st.bobId}`);
    await alice.getByLabel('Amount', { exact: true }).fill('0.07');
    await alice.getByLabel(/What is it for/).fill('Navigation budget fixture');
    await alice.getByRole('button', { name: 'Review trade', exact: true }).click();
    await alice.getByRole('button', { name: 'Send trade', exact: true }).click();
    const match = new URL(alice.url()).pathname.match(/\/trades\/([^/]+)$/);
    expect(match).not.toBeNull();
    const id = match[1];
    expect([st.tradePending, st.tradeDone, st.tradeReverse]).not.toContain(id);
    const trade = `${n(st)}/trades/${id}`;
    await bob.goto(trade + '/accept');
    await bob.getByRole('button', { name: 'Confirm and accept', exact: true }).click();
    await expect(bob.locator('#flash')).toContainText('Trade completed.');

    // The measured path begins here. No direct goto or form submission occurs
    // inside it: Home -> History -> this trade -> Raise a dispute.
    await alice.setViewportSize({ width: viewport.width, height: viewport.height });
    await startAtHome(alice, st);
    await followPath(alice, st, [history(st), trade, trade + '/dispute']);
    await expect(alice.locator('h1')).toHaveText('Raise a dispute');
    await expect(alice.getByRole('button', { name: 'Review dispute', exact: true })).toBeVisible();
    // Do not submit a dispute or trigger a reviewer decision/reversal.
  });
}
