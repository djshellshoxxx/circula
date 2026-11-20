// UX-028: hold real document responses so the progressive live state can be
// observed while ordinary keyboard navigation and native forms remain intact.
const { test, expect } = require('../fixtures');

async function holdRequest(page, url) {
  let release;
  let reached;
  const gate = new Promise((resolve) => { release = resolve; });
  const request = new Promise((resolve) => { reached = resolve; });
  await page.route(url, async (route) => {
    reached(route.request());
    await gate;
    await route.continue();
  });
  return { release, request };
}

test('keyboard navigation announces loading without moving focus', async ({ as, st }) => {
  const page = await as('alice');
  await page.goto(`/n/${st.slug}`);
  const history = page.getByRole('link', { name: 'History', exact: true });
  await history.focus();
  const held = await holdRequest(page, `${st.base}/n/${st.slug}/history`);
  const activation = page.keyboard.press('Enter');
  try {
    await held.request;
    await expect(page.locator('#page-loading')).toHaveText('Loading the next page. Please wait.');
    await expect(page.locator('#page-loading')).toHaveAttribute('role', 'status');
    await expect(page.locator('main')).toHaveAttribute('aria-busy', 'true');
    await expect(history).toBeFocused();
  } finally {
    held.release();
  }
  await activation;
  await expect(page).toHaveURL(`${st.base}/n/${st.slug}/history`);
  await expect(page.locator('#page-loading')).toBeEmpty();
  expect(await page.locator('main').getAttribute('aria-busy')).toBeNull();
});

test('submission feedback retains confirmation data and successful controls', async ({ as, st }) => {
  const page = await as('alice');
  const proposal = `/n/${st.slug}/trades/new`;
  await page.goto(proposal);
  await page.getByLabel('Who is the trade with?', { exact: true }).selectOption(st.bobId);
  await page.getByLabel('Amount', { exact: true }).fill('0.01');
  await page.getByRole('button', { name: 'Review trade', exact: true }).click();
  const confirm = page.getByRole('button', { name: 'Send trade', exact: true });
  await expect(confirm).toBeVisible();
  const held = await holdRequest(page, st.base + proposal);
  const activation = confirm.click();
  try {
    const request = await held.request;
    const values = new URLSearchParams(request.postData());
    expect(request.method()).toBe('POST');
    expect(values.get('step')).toBe('confirm');
    expect(values.get('amount')).toBe('0.01');
    expect(values.get('counterparty_id')).toBe(st.bobId);
    expect(values.get('csrf')).toBeTruthy();
    expect(values.get('idem')).toBeTruthy();
    await expect(page.locator('#page-loading')).toHaveText('Sending your request. Please wait.');
    await expect(confirm).toBeEnabled();
    await expect(page.locator('main')).toHaveAttribute('aria-busy', 'true');
  } finally {
    held.release();
  }
  await activation;
  await expect(page.locator('#flash')).toContainText('Trade sent to');
  await expect(page.locator('#page-loading')).toBeEmpty();
  expect(await page.locator('main').getAttribute('aria-busy')).toBeNull();
});

test('native validation does not announce a submission that was not sent', async ({ page }) => {
  await page.goto('/register');
  // The existing form normally uses server validation. Enable the browser's
  // required-field gate in this fixture to exercise native invalid submission.
  await page.locator('form[action="/register"]').evaluate((form) => { form.noValidate = false; });
  await page.getByRole('button', { name: 'Create account', exact: true }).click();
  expect(await page.getByLabel('Login', { exact: true }).evaluate((field) => field.validity.valueMissing)).toBe(true);
  await expect(page.locator('#page-loading')).toBeEmpty();
  expect(await page.locator('main').getAttribute('aria-busy')).toBeNull();
  await expect(page).toHaveURL((url) => url.pathname === '/register');
});

test('a native export download leaves its page idle and usable', async ({ as, st }) => {
  const page = await as('alice');
  const history = `/n/${st.slug}/history`;
  await page.goto(history);
  const button = page.getByRole('button', { name: 'Download my transactions', exact: true });
  const [download] = await Promise.all([page.waitForEvent('download'), button.click()]);
  expect(download.suggestedFilename()).toMatch(/\.zip$/);
  await expect(page).toHaveURL((url) => url.pathname === history);
  await expect(page.locator('#page-loading')).toBeEmpty();
  expect(await page.locator('main').getAttribute('aria-busy')).toBeNull();
  await expect(button).toBeEnabled();
});
