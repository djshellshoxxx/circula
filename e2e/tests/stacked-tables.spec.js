const { test, expect } = require('../fixtures');

// UX-037: the same audit records remain readable as labelled stacked rows.
for (const [who, suffix] of [['admin', '/admin/audit'], ['alice', '/audit']]) {
  test(`wide ${who} audit table retains every value in labelled narrow rows`, async ({ as, st }) => {
    const p = await as(who);
    await p.setViewportSize({ width: 1280, height: 900 });
    await p.goto(`/n/${st.slug}${suffix}`);
    const table = p.locator('table.stack');
    await expect(table).toHaveCount(1);
    expect(await table.locator('thead th').count()).toBeGreaterThan(4);
    expect(await table.locator('tbody td').count()).toBeGreaterThan(0);
    const before = await table.locator('tbody').textContent();
    await expect(table.locator('tbody td').first()).toHaveCSS('display', 'table-cell');
    await p.setViewportSize({ width: 320, height: 900 });
    await expect(table.locator('tbody td').first()).toHaveCSS('display', 'block');
    expect(await table.locator('tbody').textContent()).toBe(before);
    const unlabelled = await table.locator('tbody td').evaluateAll((cells) => cells
      .filter((cell) => {
        const label = cell.getAttribute('data-label');
        const content = getComputedStyle(cell, '::before').content;
        return !label || !content.includes(label) || getComputedStyle(cell).display !== 'block';
      }).map((cell) => cell.outerHTML));
    expect(unlabelled).toEqual([]);
  });
}
