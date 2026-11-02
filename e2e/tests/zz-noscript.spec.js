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
    await expect(balance).toContainText(/In debt by|In credit by|Balance is zero|Your balance is/);
    await expect(balance.getByText('Credit limit', { exact: true })).toBeVisible();
    await expect(balance.getByText('Reserved for pending outgoing trades', { exact: true })).toBeVisible();
    await expect(balance).toContainText('Available to spend');
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

test('QA-034: no-script keyboard registration and invitation acceptance preserve the return path', async ({ browser, st }) => {
  const contexts = [];
  try {
    const admin = await newPage(browser, st, contexts, 'admin');
    await admin.goto(`/n/${st.slug}/admin/invitations`);
    await expect(admin.getByLabel('Valid for (days)', { exact: true })).toHaveValue('7');
    await kbActivate(admin, /^Create invitation link$/);
    const invitation = new URL((await admin.locator('#flash code').innerText()).trim());
    expect(invitation.origin, 'invitation stays on the temporary test instance').toBe(st.base);
    expect(invitation.pathname).toMatch(/^\/join\/[^/]+$/);
    expect(invitation.pathname, 'the seeded unused invitation is not consumed').not.toBe(st.unusedInvite);

    const member = await newPage(browser, st, contexts);
    await member.goto(invitation.pathname);
    await expect(member.locator('h1')).toHaveText('Accept your invitation');
    await kbActivate(member, /^Create an account$/);
    await expect(member).toHaveURL((url) => url.pathname === '/register' && url.searchParams.get('next') === invitation.pathname);
    const login = 'noscript-invite@example.org';
    await kbSubmit(member, [[/^Login$/, login], [/^Password$/, st.password], [/^Confirm password$/, st.password]], /^Create account$/);
    await expect(member).toHaveURL((url) => url.pathname === '/login' && url.searchParams.get('next') === invitation.pathname);
    await expect(member.locator('#flash')).toContainText('Sign in with it and your password.');
    await kbSubmit(member, [[/^Login$/, login], [/^Password$/, st.password]], /^Sign in$/);
    await expect(member).toHaveURL((url) => url.pathname === invitation.pathname);
    await kbSubmit(member, [[/^Your display name$/, 'No Script Member']], /^Join the network$/);
    await expect(member).toHaveURL((url) => url.pathname.replace(/\/$/, '') === `/n/${st.slug}`);
    await expect(member.locator('#flash')).toContainText('You have joined Demo Circle.');
    await expect(member.getByRole('region', { name: 'Your balance', exact: true })).toBeVisible();
    await kbActivate(member, /^Directory$/);
    await expect(member.getByRole('link', { name: 'No Script Member', exact: true })).toBeVisible();
  } finally {
    await Promise.all(contexts.map((context) => context.close()));
  }
});

test('QA-034: no-script keyboard dispute review, opening and other-party response', async ({ browser, st }) => {
  const contexts = [];
  try {
    const alice = await newPage(browser, st, contexts, 'alice');
    await alice.goto(`/n/${st.slug}`);
    const id = await propose(alice, st, '0.03', 'no-script dispute journey');
    const bob = await newPage(browser, st, contexts, 'bob');
    await openPending(bob, st, '0.03', id);
    await kbActivate(bob, /^Review and accept$/);
    await kbActivate(bob, /^Confirm and accept$/);
    await expect(bob.locator('#flash')).toContainText('Trade completed.');

    await alice.goto(`/n/${st.slug}`);
    const balanceBefore = await alice.getByRole('region', { name: 'Your balance', exact: true }).innerText();
    await alice.goto(`/n/${st.slug}/trades/${id}`);
    await kbActivate(alice, /^Raise a dispute$/);
    await expect(alice.getByLabel('Reason', { exact: true })).toHaveValue('not_delivered');
    const statement = 'The agreed delivery is still missing.';
    await kbSubmit(alice, [[/^What went wrong\?$/, statement]], /^Review dispute$/);
    await expect(alice.locator('h1')).toHaveText('Review your dispute');
    await expect(alice.locator('main')).toContainText(statement);
    await expect(alice.locator('main')).toContainText('Your balance and available credit do not change until a reviewer decides.');
    await kbActivate(alice, /^Open dispute$/);
    await expect(alice.locator('#flash')).toContainText('Dispute opened. The trade is marked Disputed. Your balance has not changed.');
    const disputePath = new URL(alice.url()).pathname;
    expect(disputePath).toMatch(new RegExp(`^/n/${st.slug}/disputes/[^/]+$`));
    expect([st.disputePath, st.disputeResolved, st.disputeReReview]).not.toContain(disputePath);
    await expect(alice.locator('main')).toContainText(statement);

    await bob.goto(disputePath);
    const response = 'I will contact you to arrange the delivery.';
    await kbSubmit(bob, [[/^Your response$/, response]], /^Send response$/);
    await expect(bob.locator('#flash')).toContainText('Your response was added');
    await expect(bob.locator('main')).toContainText(response);
    await expect(bob.getByRole('button', { name: 'Send response', exact: true })).toHaveCount(0);

    await alice.goto(disputePath);
    await expect(alice.locator('main')).toContainText(response);
    await kbActivate(alice, /^Home$/);
    await expect(alice.getByRole('region', { name: 'Your balance', exact: true })).toHaveText(balanceBefore, { useInnerText: true });
    await expect(await historyRow(alice, st, id)).toContainText('Disputed');
  } finally {
    await Promise.all(contexts.map((context) => context.close()));
  }
});

test('QA-034/UX-022: no-script keyboard validation links retain and correct a proposal', async ({ browser, st }) => {
  const contexts = [];
  try {
    const page = await newPage(browser, st, contexts, 'alice');
    await page.goto(`/n/${st.slug}/trades/new?to=${st.bobId}`);
    const note = 'Correct this no-script preview without sending it.';
    await kbSubmit(page, [[/^Amount$/, 'not an amount'], [/^What is it for/, note]], /^Review trade$/);
    const summary = page.locator('#error-summary');
    await expect(summary).toBeVisible();
    await expect(summary).toHaveAttribute('role', 'alert');
    const amount = page.getByLabel('Amount', { exact: true });
    await expect(amount).toHaveValue('not an amount');
    await expect(amount).toHaveAttribute('aria-invalid', 'true');
    await expect(page.getByLabel(/What is it for/)).toHaveValue(note);
    // Native fragment navigation must work without the site's focus script.
    await kbActivate(page, /^Amount:/);
    await expect(amount).toBeFocused();
    const ring = await amount.evaluate((el) => {
      const cs = getComputedStyle(el);
      return (cs.outlineStyle !== 'none' && parseFloat(cs.outlineWidth) >= 2) || cs.boxShadow !== 'none';
    });
    expect(ring, 'keyboard-corrected amount has a visible focus indicator').toBe(true);
    await page.keyboard.press('ControlOrMeta+A');
    await page.keyboard.type('0.04');
    await kbActivate(page, /^Review trade$/);
    await expect(page.locator('#error-summary')).toHaveCount(0);
    await expect(page.locator('h1')).toHaveText('Review your trade');
    await expect(page.locator('main')).toContainText(note);
    await expect(page.getByRole('button', { name: 'Send trade', exact: true })).toBeVisible();
    // Do not send: this case validates correction/preview, not a new posting.
  } finally {
    await Promise.all(contexts.map((context) => context.close()));
  }
});
