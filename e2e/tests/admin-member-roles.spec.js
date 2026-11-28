const { test, expect, expectNoAxeViolations } = require('../fixtures');

test('member roles and adjacent actions remain usable on a narrow screen', async ({ as, st }) => {
  const page = await as('admin');
  await page.setViewportSize({ width: 320, height: 850 });
  await page.goto(`/n/${st.slug}/admin/members`);

  const roles = (name) => page.getByRole('row')
    .filter({ has: page.getByRole('rowheader', { name, exact: true }) })
    .locator('td[data-label="Roles"]');
  await expect(roles('Dana Admin').getByRole('listitem')).toHaveText(['admin']);
  await expect(roles('Carol Member').getByRole('listitem')).toHaveText(['reviewer']);
  await expect(roles('Alice Member')).toHaveText('member');
  await expect(roles('Alice Member').getByRole('list')).toHaveCount(0);
  const actions = page.locator('table.stack tbody td:last-child a');
  expect(await actions.count()).toBeGreaterThan(0);
  const undersized = await actions.evaluateAll((links) => links.filter((link) => {
    const { width, height } = link.getBoundingClientRect();
    return width < 24 || height < 24;
  }).map((link) => link.getAttribute('aria-label')));
  expect(undersized).toEqual([]);
  await expectNoAxeViolations(page, 'narrow member list');
});
