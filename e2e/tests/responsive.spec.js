// UX-033 / UX-036: representative core screens in portrait and landscape.
// These tests navigate seeded pages only; they do not post trades or change policy.
const { test, expect } = require('../fixtures');

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
];

for (const viewport of VIEWPORTS) {
  for (const [name, who, pathOf, expectedPathOf = pathOf] of PAGES) {
    test(`UX-033/UX-036: ${name}, ${viewport.name}`, async ({ st, as, page: anon }) => {
      const page = who ? await as(who) : anon;
      await page.setViewportSize({ width: viewport.width, height: viewport.height });
      const path = pathOf(st);
      const response = await page.goto(path);
      expect(response, `${name}: navigation response`).not.toBeNull();
      expect(response.status(), `${name}: status`).toBe(200);
      const expectedPath = new URL(expectedPathOf(st), st.base).pathname.replace(/\/$/, '');
      await expect(page).toHaveURL((url) => url.pathname.replace(/\/$/, '') === expectedPath);
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
    });
  }
}
