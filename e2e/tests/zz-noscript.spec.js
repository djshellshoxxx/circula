// QA-034: keyboard-only member journeys with site JavaScript disabled.
// Uses only newly proposed trades; seeded trade IDs and policies are untouched.
const path = require('path');
const { test, expect, STATE_DIR } = require('../fixtures');
const { tabTo, kbSubmit, kbActivate } = require('../kbd');

async function newPage(browser, st, contexts, who) {
  const options = { baseURL: st.base, javaScriptEnabled: false };
  if (who) options.storageState = path.join(STATE_DIR, who + '.json');
  const context = await browser.newContext(options);
  contexts.push(context);
  return context.newPage();
}

async function propose(page, st, amount, note) {
  await kbActivate(page, /^Propose a trade$/);
  await tabTo(page, /^Who is the trade with\?$/);
  await page.keyboard.type('Bob');
  await expect(page.getByLabel('Who is the trade with?', { exact: true })).toHaveValue(st.bobId);
  await expect(page.getByLabel('I pay them', { exact: true })).toBeChecked();
  await kbSubmit(page, [[/^Amount$/, amount], [/^What is it for/, note]], /^Review trade$/);
  await expect(page.locator('h1')).toHaveText(/Review/i);
  await kbActivate(page, /^Send trade$/);
  await expect(page.locator('#flash')).toContainText('Trade sent to');
  const match = new URL(page.url()).pathname.match(/\/trades\/([^/]+)$/);
  expect(match, 'proposal lands on its own trade detail').not.toBeNull();
  return match[1];
}

async function openPending(page, st, amount, id) {
  await page.goto(`/n/${st.slug}`);
  await kbActivate(page, new RegExp(amount.replace('.', '\\.')));
  await expect(page).toHaveURL((url) => url.pathname === `/n/${st.slug}/trades/${id}`);
}

async function historyRow(page, st, id) {
  await kbActivate(page, /^History$/);
  await expect(page).toHaveURL((url) => url.pathname === `/n/${st.slug}/history`);
  return page.locator('main li').filter({
    has: page.locator(`a[href="/n/${st.slug}/trades/${id}"]`),
  });
}

test('QA-034: no-script keyboard sign-in, proposal, acceptance, balance and history', async ({ browser, st }) => {
  const contexts = [];
  try {
    const alice = await newPage(browser, st, contexts);
    await alice.goto('/login');
    await kbSubmit(alice, [[/^Login$/, st.users.alice], [/^Password$/, st.password]], /^Sign in$/);
    await expect(alice).toHaveURL((url) => url.pathname.replace(/\/$/, '') === `/n/${st.slug}`);

    const id = await propose(alice, st, '0.01', 'no-script keyboard acceptance');
    const bob = await newPage(browser, st, contexts, 'bob');
    await openPending(bob, st, '0.01', id);
    await kbActivate(bob, /^Review and accept$/);
    await expect(bob.getByRole('button', { name: 'Confirm and accept', exact: true })).toBeVisible();
    await kbActivate(bob, /^Confirm and accept$/);
    await expect(bob.locator('#flash')).toContainText('Trade completed.');
    await expect(await historyRow(bob, st, id)).toContainText('Completed');

    await kbActivate(bob, /^Home$/);
    const balance = bob.getByRole('region', { name: 'Your balance', exact: true });
    await expect(balance).toBeVisible();
    await expect(balance).toContainText('Available to spend');
    await expect(bob.getByRole('meter', { name: 'Credit limit in use', exact: true })).toBeVisible();
  } finally {
    await Promise.all(contexts.map((context) => context.close()));
  }
});

test('QA-034: no-script keyboard proposal and decline with no posting', async ({ browser, st }) => {
  const contexts = [];
  try {
    const alice = await newPage(browser, st, contexts, 'alice');
    await alice.goto(`/n/${st.slug}`);
    const id = await propose(alice, st, '0.02', 'no-script keyboard decline');
    const bob = await newPage(browser, st, contexts, 'bob');
    await openPending(bob, st, '0.02', id);
    await kbActivate(bob, /^Decline$/);
    await expect(bob.locator('h1')).toHaveText('Decline this trade?');
    await kbActivate(bob, /^Decline this trade$/);
    await expect(bob.locator('#flash')).toContainText('Trade declined. Nothing was posted.');
    await expect(bob.getByText('Ledger reference', { exact: true })).toHaveCount(0);
    await expect(await historyRow(alice, st, id)).toContainText('Declined');
  } finally {
    await Promise.all(contexts.map((context) => context.close()));
  }
});
