// UX-033 / UX-036: representative core screens in portrait and landscape.
// These tests navigate seeded pages only; they do not post trades or change policy.
// Rows whose role is "apply:<name>" use the approval-mode instance (apply.js).
const { test, expect } = require('../apply');

const VIEWPORTS = [
  { name: '320px portrait', width: 320, height: 640 },
  { name: '640px landscape', width: 640, height: 320 },
];

const PAGES = [
  ['login', null, () => '/login'],
  ['registration', null, () => '/register'],
  ['help', null, () => '/help'],
  ['dashboard', 'alice', (st) => `/n/${st.slug}`],
  ['directory', 'alice', (st) => `/n/${st.slug}/directory`],
  ['new listing', 'alice', (st) => `/n/${st.slug}/listings/new`],
  ['propose trade', 'alice', (st) => `/n/${st.slug}/trades/new?to=${st.bobId}`],
  ['completed trade', 'alice', (st) => `/n/${st.slug}/trades/${st.tradeDone}`],
  ['history', 'alice', (st) => `/n/${st.slug}/history`],
  ['admin overview', 'admin', (st) => `/n/${st.slug}/admin`],
  ['admin members', 'admin', (st) => `/n/${st.slug}/admin/members`],
  ['admin audit', 'admin', (st) => `/n/${st.slug}/admin/audit`],
  ['pending trade (recipient)', 'bob', (st) => `/n/${st.slug}/trades/${st.tradePending}`],
  ['accept a trade (recipient)', 'bob', (st) => `/n/${st.slug}/trades/${st.tradePending}/accept`],
  ['decline a trade (recipient)', 'bob', (st) => `/n/${st.slug}/trades/${st.tradePending}/decline`],
  ['cancel own proposal', 'carol', (st) => `/n/${st.slug}/trades/${st.tradePending}/cancel`],
  ['invitation (signed out)', null, (st) => st.unusedInvite],
  ['invitation (signed in)', 'alice', (st) => st.unusedInvite],
  ['home (single-network redirect)', 'carol', () => '/', (st) => `/n/${st.slug}`],
  ['create network form', 'carol', () => '/setup'],
  ['password change form', 'alice', () => '/account/password'],
  ['help (member)', 'alice', () => '/help'],
  ['dashboard (recipient)', 'bob', (st) => `/n/${st.slug}`],
  ['dashboard (reviewer with announcement)', 'carol', (st) => `/n/${st.slug}`],
  ['member profile', 'alice', (st) => st.bobProfile],
  ['listings', 'bob', (st) => `/n/${st.slug}/listings`],
  ['edit own listing', 'alice', (st) => st.listingEdit],
  ['trades', 'alice', (st) => `/n/${st.slug}/trades`],
  ['inbox (member)', 'alice', (st) => `/n/${st.slug}/inbox`],
  ['inbox (recipient)', 'bob', (st) => `/n/${st.slug}/inbox`],
  ['inbox (unread filter)', 'carol', (st) => `/n/${st.slug}/inbox?show=unread`],
  ['report a listing', 'carol', (st) => st.reportPath],
  ['report a member', 'alice', (st) => `/n/${st.slug}/report?type=member&id=${st.bobId}`],
  ['time zone form', 'alice', () => '/account/time-zone'],
  ['raise a dispute form', 'bob', (st) => `/n/${st.slug}/trades/${st.tradeDone}/dispute`],
  ['disputes', 'alice', (st) => `/n/${st.slug}/disputes`],
  ['dispute (opener)', 'alice', (st) => st.disputePath],
  ['dispute (other party)', 'carol', (st) => st.disputePath],
  ['resolved dispute (opener)', 'alice', (st) => st.disputeResolved],
  ['resolved dispute (other party)', 'bob', (st) => st.disputeResolved],
  ['re-review requested (requester)', 'alice', (st) => st.disputeReReview],
  ['re-review requested (independent reviewer)', 'carol', (st) => st.disputeReReview],
  ['admin suspend member form', 'admin', (st) => `/n/${st.slug}/admin/members/${st.bobId}/suspend`],
  ['admin invitations', 'admin', (st) => `/n/${st.slug}/admin/invitations`],
  ['admin policy form', 'admin', (st) => `/n/${st.slug}/admin/policy`],
  ['admin adjustment form', 'admin', (st) => `/n/${st.slug}/admin/adjust`],
  ['admin exports', 'admin', (st) => `/n/${st.slug}/admin/exports`],
  ['admin reports (open filter)', 'admin', (st) => `/n/${st.slug}/admin/reports`],
  ['admin reports (all filter)', 'admin', (st) => `/n/${st.slug}/admin/reports?status=all`],
  ['admin announcements', 'admin', (st) => `/n/${st.slug}/admin/announcements`],
  ['admin reversal form', 'admin', (st) => `/n/${st.slug}/admin/reverse`],
  ['admin reversal form (trade chosen)', 'admin', (st) => `/n/${st.slug}/admin/reverse?tx=${st.tradeReverse}`],
  ['review queue', 'admin', (st) => `/n/${st.slug}/review`],
  ['dispute (reviewer)', 'admin', (st) => st.disputePath],
  ['re-review requested (original reviewer)', 'admin', (st) => st.disputeReReview],
  ['account recovery (signed out)', null, () => '/recover'],
  ['account security', 'alice', () => '/account/security'],
  ['two-step sign-in setup', 'alice', () => '/account/security/mfa/enrol'],
  ['password recovery codes', 'alice', () => '/account/recovery-codes'],
  ['network rules and change log', 'alice', (st) => `/n/${st.slug}/policy`],
  ['network terms (none published)', 'alice', (st) => `/n/${st.slug}/terms`],
  ['leave the network (blocked)', 'alice', (st) => `/n/${st.slug}/leave`],
  ['own account activity log', 'alice', (st) => `/n/${st.slug}/audit`],
  ['admin inactive members', 'admin', (st) => `/n/${st.slug}/admin/inactive`],
  ['admin adjustments awaiting approval', 'admin', (st) => `/n/${st.slug}/admin/adjustments/pending`],
  ['admin pause or close', 'admin', (st) => `/n/${st.slug}/admin/network`],
  ['admin operator access', 'admin', (st) => `/n/${st.slug}/admin/break-glass`],
  ['admin network terms (none published)', 'admin', (st) => `/n/${st.slug}/admin/terms`],
  ['admin set credit limit form', 'admin', (st) => `/n/${st.slug}/admin/members/${st.bobId}/limit`],
  ['network terms (published, to accept)', 'apply:hana', (st) => `/n/${st.apply.slug}/terms`],
  ['network rules (with terms)', 'apply:hana', (st) => `/n/${st.apply.slug}/policy`],
  ['leave the network (form)', 'apply:hana', (st) => `/n/${st.apply.slug}/leave`],
  ['admin overview (requests waiting)', 'apply:erin', (st) => `/n/${st.apply.slug}/admin`],
  ['admin membership requests', 'apply:erin', (st) => `/n/${st.apply.slug}/admin/members?status=pending`],
  ['admin approve a request', 'apply:erin', (st) => `/n/${st.apply.slug}/admin/members/${st.apply.gailId}/approve`],
  ['admin deny a request', 'apply:erin', (st) => `/n/${st.apply.slug}/admin/members/${st.apply.gailId}/deny`],
  ['admin network terms (published)', 'apply:erin', (st) => `/n/${st.apply.slug}/admin/terms`],
  ['join by request (signed in)', 'apply:gail', (st) => `/n/${st.apply.slug}/join`],
];

for (const viewport of VIEWPORTS) {
  for (const [name, who, pathOf, expectedPathOf = pathOf] of PAGES) {
    test(`UX-033/UX-036: ${name}, ${viewport.name}`, async ({ st, open, page: anon }) => {
      const page = who ? await open(who) : anon;
      await page.setViewportSize({ width: viewport.width, height: viewport.height });
      const path = pathOf(st);
      const response = await page.goto(path);
      expect(response, `${name}: navigation response`).not.toBeNull();
      expect(response.status(), `${name}: status`).toBe(200);
      const expectedPath = new URL(expectedPathOf(st), st.base).pathname.replace(/\/$/, '');
      await expect(page).toHaveURL((url) => url.pathname.replace(/\/$/, '') === expectedPath);
      await expectReflow(page, name, viewport);
    });
  }
}

/** UX-033: one h1 in a visible main, no page-level horizontal overflow, and scrolling tables in labelled focusable regions. */
async function expectReflow(page, name, viewport) {
  await expect(page.locator('main')).toBeVisible();
  await expect(page.locator('h1')).toHaveCount(1);
  await expect(page.locator('h1')).toBeVisible();

  const layout = await page.evaluate(() => {
    const root = document.documentElement;
    const tableIssues = [];
    for (const table of document.querySelectorAll('table')) {
      // Horizontal scrolling is allowed inside a labelled, focusable table region.
      for (let parent = table.parentElement; parent && parent !== document.body; parent = parent.parentElement) {
        const style = getComputedStyle(parent);
        if (parent.scrollWidth <= parent.clientWidth || !['auto', 'scroll'].includes(style.overflowX)) continue;
        const labelledBy = (parent.getAttribute('aria-labelledby') || '').split(/\s+/)
          .map((id) => document.getElementById(id)?.textContent || '').join(' ').trim();
        const label = (parent.getAttribute('aria-label') || '').trim() || labelledBy;
        if (parent.getAttribute('role') !== 'region' || parent.tabIndex < 0 || !label) {
          tableIssues.push(table.caption?.textContent.trim() || 'unnamed table');
        }
        break;
      }
    }
    return {
      viewportWidth: root.clientWidth,
      documentWidth: Math.max(root.scrollWidth, document.body.scrollWidth),
      tableIssues,
    };
  });

  expect(layout.viewportWidth, `${name}: viewport is present`).toBeGreaterThan(0);
  expect(layout.documentWidth, `${name}: page-level horizontal overflow at ${viewport.name}`)
    .toBeLessThanOrEqual(layout.viewportWidth);
  expect(layout.tableIssues, `${name}: scrolling tables need a labelled keyboard-focusable region`)
    .toEqual([]);
}

// One-time pages that only a form post shows. Each uses a fresh account or a read-only check, so the
// shared seed keeps its state.
for (const viewport of VIEWPORTS) {
  test(`UX-033/UX-036: two-step sign-in setup key and password recovery codes (fresh account), ${viewport.name}`, async ({ page, st }) => {
    const login = `reflow-${viewport.width}@example.org`;
    await page.setViewportSize({ width: viewport.width, height: viewport.height });
    await page.goto('/register');
    await page.getByLabel('Login', { exact: true }).fill(login);
    await page.getByLabel('Password', { exact: true }).fill(st.password);
    await page.getByLabel('Confirm password', { exact: true }).fill(st.password);
    await page.getByRole('button', { name: 'Create account', exact: true }).click();
    await expect(page).toHaveURL((url) => url.pathname === '/login');
    await page.getByLabel('Login', { exact: true }).fill(login);
    await page.getByLabel('Password', { exact: true }).fill(st.password);
    await page.getByRole('button', { name: 'Sign in', exact: true }).click();
    await expect(page).toHaveURL((url) => url.pathname === '/');

    await page.goto('/account/security/mfa/enrol');
    await page.getByLabel('Password', { exact: true }).fill(st.password);
    await page.getByRole('button', { name: 'Start setup', exact: true }).click();
    await expect(page.getByLabel('Code from your app', { exact: true })).toBeVisible();
    await expect(page.locator('main code').first()).toBeVisible();
    await expectReflow(page, 'two-step sign-in setup key', viewport);

    await page.goto('/account/recovery-codes');
    await page.getByLabel('Current password', { exact: true }).fill(st.password);
    await page.getByRole('button', { name: 'Make new recovery codes', exact: true }).click();
    await expect(page.locator('ol.codes li')).toHaveCount(10);
    await expectReflow(page, 'new password recovery codes', viewport);
  });

  test(`UX-033/UX-036: ledger consistency check result, ${viewport.name}`, async ({ asApply, st }) => {
    const page = await asApply('erin');
    await page.setViewportSize({ width: viewport.width, height: viewport.height });
    await page.goto(`/n/${st.apply.slug}/admin/audit`);
    await page.getByRole('button', { name: 'Run consistency check', exact: true }).click();
    await expect(page).toHaveURL((url) => url.pathname === `/n/${st.apply.slug}/admin/check`);
    await expect(page.getByRole('link', { name: 'Back to audit and checks', exact: true })).toBeVisible();
    await expectReflow(page, 'ledger check result', viewport);
  });
}

for (const viewport of VIEWPORTS) {
  test(`UX-028/UX-033: not-found state and Help recovery, ${viewport.name}`, async ({ page }) => {
    await page.setViewportSize({ width: viewport.width, height: viewport.height });
    const response = await page.goto('/no-such-page');
    expect(response).not.toBeNull();
    expect(response.status(), 'the error page preserves its 404 status').toBe(404);
    await expect(page).toHaveURL((url) => url.pathname === '/no-such-page');
    await expect(page.locator('main')).toBeVisible();
    await expect(page.locator('h1')).toHaveCount(1);
    await expect(page.locator('h1')).toBeVisible();
    const layout = await page.evaluate(() => ({
      viewport: document.documentElement.clientWidth,
      document: Math.max(document.documentElement.scrollWidth, document.body.scrollWidth),
    }));
    expect(layout.viewport).toBeGreaterThan(0);
    expect(layout.document, 'not-found content stays within the viewport').toBeLessThanOrEqual(layout.viewport);
    const help = page.getByRole('link', { name: 'Help: credits, balances and statuses', exact: true });
    await expect(help).toHaveAttribute('href', '/help');
    await help.click();
    await expect(page).toHaveURL((url) => url.pathname === '/help');
    await expect(page.locator('h1')).toBeVisible();
  });
}
