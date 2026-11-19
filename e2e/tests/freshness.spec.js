const { test, expect } = require('../fixtures');

// PER-022: connectivity loss never presents retained figures as refreshed.
test('balance snapshot retains its time through network failure and reconnection', async ({ as, st }) => {
  const p = await as('alice');
  await p.goto(`/n/${st.slug}`);
  const freshness = p.locator('#figure-freshness');
  const original = await freshness.textContent();
  const time = await freshness.locator('time').getAttribute('datetime');
  expect(time).toBeTruthy();
  await expect(freshness).toContainText('do not update automatically');
  await expect(p.locator('#figure-stale')).toBeHidden();
  const figures = await p.getByRole('region', { name: 'Your balance' }).textContent();
  await p.context().setOffline(true);
  await expect(p.locator('#figure-stale')).toBeVisible();
  await expect(p.locator('#figure-stale')).toContainText('connection is unavailable');
  await expect(p.locator('#figure-stale')).toContainText(original);
  await expect(p.getByRole('region', { name: 'Your balance' })).toHaveText(figures);
  await p.context().setOffline(false);
  // Restored connectivity alone must not erase a stale-data indication.
  await expect(p.locator('#figure-stale')).toBeVisible();
  await expect(freshness.locator('time')).toHaveAttribute('datetime', time);
  await p.reload();
  await expect(p.locator('#figure-stale')).toBeHidden();
  await expect(p.locator('#figure-freshness')).toBeVisible();
});

test('restoring a balance page from history identifies its earlier snapshot', async ({ as, st }) => {
  const p = await as('alice');
  await p.goto(`/n/${st.slug}`);
  const original = await p.locator('#figure-freshness').textContent();
  await p.evaluate(() => window.dispatchEvent(new PageTransitionEvent('pageshow', { persisted: true })));
  await expect(p.locator('#figure-stale')).toContainText('restored from browser history');
  await expect(p.locator('#figure-stale')).toContainText(original);
});
