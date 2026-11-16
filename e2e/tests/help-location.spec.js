// UX-049: the shared Help link keeps the same landmark within each section.
const { test, expect } = require('../fixtures');

const VIEWPORTS = [
  { name: 'portrait', width: 320, height: 640 },
  { name: 'landscape', width: 640, height: 320 },
];

const SECTIONS = [
  ['public', null, [() => '/login', () => '/register', () => '/help']],
  ['member', 'alice', [
    (st) => `/n/${st.slug}`, (st) => `/n/${st.slug}/directory`,
    (st) => `/n/${st.slug}/listings/new`, (st) => `/n/${st.slug}/trades/new`,
    (st) => `/n/${st.slug}/trades/${st.tradeDone}`, (st) => `/n/${st.slug}/history`,
  ]],
  ['administrator', 'admin', [
    (st) => `/n/${st.slug}/admin`, (st) => `/n/${st.slug}/admin/members`,
    (st) => `/n/${st.slug}/admin/audit`,
  ]],
];

for (const viewport of VIEWPORTS) {
  for (const [section, who, paths] of SECTIONS) {
    test(`UX-049: shared Help landmark, ${section}, ${viewport.name}`, async ({ st, as, page: anon }) => {
      const page = who ? await as(who) : anon;
      await page.setViewportSize({ width: viewport.width, height: viewport.height });
      let expectedPlacement;
      for (const pathOf of paths) {
        const path = pathOf(st);
        const response = await page.goto(path);
        expect(response, `${path}: response`).not.toBeNull();
        expect(response.status(), `${path}: status`).toBe(200);
        const expectedPath = new URL(path, st.base).pathname.replace(/\/$/, '');
        await expect(page).toHaveURL((url) => url.pathname.replace(/\/$/, '') === expectedPath);

        const help = page.getByRole('link', { name: 'Help: credits, balances and statuses', exact: true });
        await expect(help).toHaveCount(1);
        await expect(help).toBeVisible();
        await expect(help).toHaveAttribute('href', '/help');
        const placement = await help.evaluate((el) => {
          const landmark = el.closest('header, footer, nav, main, [role="banner"], [role="contentinfo"], [role="navigation"], [role="main"]');
          if (!landmark) return null;
          const implicit = { HEADER: 'banner', FOOTER: 'contentinfo', NAV: 'navigation', MAIN: 'main' };
          const labelledBy = (landmark.getAttribute('aria-labelledby') || '').split(/\s+/)
            .map((id) => document.getElementById(id)?.textContent || '').join(' ').trim();
          return {
            role: landmark.getAttribute('role') || implicit[landmark.tagName],
            name: (landmark.getAttribute('aria-label') || '').trim() || labelledBy,
          };
        });
        expect(placement, `${path}: Help belongs to a semantic landmark`).not.toBeNull();
        expect(['banner', 'contentinfo', 'navigation', 'main']).toContain(placement.role);
        if (!expectedPlacement) expectedPlacement = placement;
        expect(placement, `${path}: Help keeps its section's landmark`).toEqual(expectedPlacement);
      }
    });
  }
}
