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
];

for (const viewport of VIEWPORTS) {
  for (const [name, who, pathOf] of PAGES) {
    test(`UX-033/UX-036: ${name}, ${viewport.name}`, async ({ st, as, page: anon }) => {
      const page = who ? await as(who) : anon;
      await page.setViewportSize({ width: viewport.width, height: viewport.height });
      const path = pathOf(st);
      const response = await page.goto(path);
      expect(response, `${name}: navigation response`).not.toBeNull();
      expect(response.status(), `${name}: status`).toBe(200);
      const expectedPath = new URL(path, st.base).pathname.replace(/\/$/, '');
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
