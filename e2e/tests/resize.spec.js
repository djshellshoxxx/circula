// UX-034: user text preferences must leave the proposal review flow usable.
// Stop at review: these tests never click Send trade or post ledger entries.
const { test, expect } = require('../fixtures');

const VIEWPORTS = [
  { name: 'portrait', width: 320, height: 640 },
  { name: 'landscape', width: 640, height: 320 },
];

const PREFERENCES = ['200% text', 'text spacing'];

async function applyPreference(page, preference, rootFontSize) {
  const css = preference === '200% text'
    ? `html { font-size: ${rootFontSize * 2}px !important; }`
    : '* { line-height: 1.5 !important; letter-spacing: 0.12em !important; word-spacing: 0.16em !important; }\n' +
      'p { margin-bottom: 2em !important; }';
  await page.addStyleTag({ content: css });
}

async function fontSize(locator) {
  return locator.evaluate((el) => parseFloat(getComputedStyle(el).fontSize));
}

for (const viewport of VIEWPORTS) {
  for (const preference of PREFERENCES) {
    test(`UX-034: proposal review with ${preference}, ${viewport.name}`, async ({ as, st }) => {
      const page = await as('alice');
      await page.setViewportSize({ width: viewport.width, height: viewport.height });
      const response = await page.goto(`/n/${st.slug}/trades/new?to=${st.bobId}`);
      expect(response, 'proposal form response').not.toBeNull();
      expect(response.status()).toBe(200);

      const amount = page.getByLabel('Amount', { exact: true });
      const review = page.getByRole('button', { name: 'Review trade', exact: true });
      const heading = page.locator('h1');
      const rootFontSize = await fontSize(page.locator('html'));
      const before = await Promise.all([fontSize(heading), fontSize(amount), fontSize(review)]);
      await applyPreference(page, preference, rootFontSize);

      if (preference === '200% text') {
        const after = await Promise.all([fontSize(heading), fontSize(amount), fontSize(review)]);
        for (let i = 0; i < before.length; i++) {
          expect(after[i], 'heading, input and button text doubles').toBeCloseTo(before[i] * 2, 1);
        }
      } else {
        const spacing = await amount.evaluate((el) => {
          const s = getComputedStyle(el);
          return { font: parseFloat(s.fontSize), line: parseFloat(s.lineHeight), letter: parseFloat(s.letterSpacing), word: parseFloat(s.wordSpacing) };
        });
        expect(spacing.line).toBeCloseTo(spacing.font * 1.5, 1);
        expect(spacing.letter).toBeCloseTo(spacing.font * 0.12, 1);
        expect(spacing.word).toBeCloseTo(spacing.font * 0.16, 1);
        const paragraph = page.locator('footer p').first();
        const margin = await paragraph.evaluate((el) => {
          const s = getComputedStyle(el);
          return { font: parseFloat(s.fontSize), bottom: parseFloat(s.marginBottom) };
        });
        expect(margin.bottom).toBeCloseTo(margin.font * 2, 1);
      }

      await expect(heading).toBeVisible();
      await amount.fill('1.00');
      await expect(amount).toHaveValue('1.00');
      await expect(review).toBeEnabled();
      await review.click();
      const send = page.getByRole('button', { name: 'Send trade', exact: true });
      await expect(send).toBeVisible();
      const sendFontSize = await fontSize(send);

      // A navigation creates a new document; reapply the same user preference.
      await applyPreference(page, preference, rootFontSize);
      if (preference === '200% text') {
        expect(await fontSize(send), 'confirmation control text doubles').toBeCloseTo(sendFontSize * 2, 1);
      }
      await expect(page.locator('h1')).toHaveText(/Review/i);
      await send.scrollIntoViewIfNeeded();
      await expect(send).toBeVisible();
      await expect(send).toBeEnabled();
    });
  }
}

// Apply the preference to every new document and verify the actual control text,
// rather than merely checking that a style tag was inserted.
async function preferControls(page, preference, controls) {
  const before = await Promise.all(controls.map(fontSize));
  await applyPreference(page, preference, await fontSize(page.locator('html')));
  for (let i = 0; i < controls.length; i++) {
    const control = controls[i];
    if (preference === '200% text') {
      expect(await fontSize(control), 'control text doubles').toBeCloseTo(before[i] * 2, 1);
    } else {
      const s = await control.evaluate((el) => {
        const cs = getComputedStyle(el);
        return { font: parseFloat(cs.fontSize), line: parseFloat(cs.lineHeight), letter: parseFloat(cs.letterSpacing), word: parseFloat(cs.wordSpacing) };
      });
      expect(s.line).toBeCloseTo(s.font * 1.5, 1);
      expect(s.letter).toBeCloseTo(s.font * 0.12, 1);
      expect(s.word).toBeCloseTo(s.font * 0.16, 1);
    }
    await control.scrollIntoViewIfNeeded();
    await expect(control).toBeVisible();
  }
}

async function preferredAction(page, preference, control) {
  await preferControls(page, preference, [page.locator('h1'), control]);
  await control.click();
}

async function preferredProposal(page, st, preference, amount, note) {
  await page.goto(`/n/${st.slug}/trades/new?to=${st.bobId}`);
  const input = page.getByLabel('Amount', { exact: true });
  const description = page.getByLabel(/What is it for/);
  const review = page.getByRole('button', { name: 'Review trade', exact: true });
  await preferControls(page, preference, [page.locator('h1'), input, description, review]);
  await input.fill(amount);
  await description.fill(note);
  await review.click();
  await expect(page.locator('h1')).toHaveText('Review your trade');
  await preferredAction(page, preference, page.getByRole('button', { name: 'Send trade', exact: true }));
  await preferControls(page, preference, [page.locator('h1')]);
  await expect(page.locator('#flash')).toContainText('Trade sent to');
  const match = new URL(page.url()).pathname.match(/\/trades\/([^/]+)$/);
  expect(match).not.toBeNull();
  return match[1];
}

for (const viewport of VIEWPORTS) {
  for (const preference of PREFERENCES) {
    test(`UX-034: registration and explicit invitation join with ${preference}, ${viewport.name}`, async ({ page, as, st }) => {
      // Use a fresh invitation and login for this variant; never consume st.unusedInvite.
      const admin = await as('admin');
      await admin.goto(`/n/${st.slug}/admin/invitations`);
      await admin.getByRole('button', { name: 'Create invitation link', exact: true }).click();
      const invitation = new URL((await admin.locator('#flash code').innerText()).trim());
      expect(invitation.origin).toBe(st.base);
      expect(invitation.pathname).toMatch(/^\/join\/[^/]+$/);
      expect(invitation.pathname).not.toBe(st.unusedInvite);
      await page.setViewportSize({ width: viewport.width, height: viewport.height });
      // Separate accepted paths: plain registration, then an explicit invitation.
      // The original no-script return-link regression stays active and failing;
      // these cases do not assert that its reserved encoding defect is repaired.
      await page.goto('/register');

      const login = `pref-${viewport.name}-${preference === '200% text' ? 'size' : 'space'}@example.org`;
      const loginField = page.getByLabel('Login', { exact: true });
      const password = page.getByLabel('Password', { exact: true });
      const confirm = page.getByLabel('Confirm password', { exact: true });
      const create = page.getByRole('button', { name: 'Create account', exact: true });
      await preferControls(page, preference, [page.locator('h1'), loginField, password, confirm, create]);
      await loginField.fill(login);
      await password.fill(st.password);
      await confirm.fill(st.password);
      await create.click();
      await expect(page).toHaveURL((url) => url.pathname === '/login' && !url.searchParams.has('next'));
      const signIn = page.getByRole('button', { name: 'Sign in', exact: true });
      await preferControls(page, preference, [page.locator('h1'), loginField, password, signIn]);
      await loginField.fill(login);
      await password.fill(st.password);
      await signIn.click();
      await expect(page).toHaveURL((url) => url.pathname === '/');
      await preferControls(page, preference, [page.locator('h1')]);
      await expect(page.locator('main')).toContainText('You are not a member of any network yet.');
      await page.goto(invitation.pathname);
      await expect(page).toHaveURL((url) => url.pathname === invitation.pathname);

      const display = page.getByLabel('Your display name', { exact: true });
      const join = page.getByRole('button', { name: 'Join the network', exact: true });
      await preferControls(page, preference, [page.locator('h1'), display, join]);
      await display.fill(`Text ${viewport.name} ${preference === '200% text' ? 'size' : 'space'}`);
      await join.click();
      await expect(page).toHaveURL((url) => url.pathname.replace(/\/$/, '') === `/n/${st.slug}`);
      await expect(page.locator('#flash')).toContainText('You have joined Demo Circle.');
      const balance = page.getByRole('region', { name: 'Your balance', exact: true });
      const history = page.getByRole('link', { name: 'History', exact: true });
      await preferControls(page, preference, [page.locator('h1'), balance, history]);
      await expect(balance).toContainText('Available to spend');
      await history.click();
      await expect(page).toHaveURL((url) => url.pathname === `/n/${st.slug}/history`);
      await preferControls(page, preference, [page.locator('h1')]);
      await expect(page.getByRole('region', { name: 'Posted and closed', exact: true })).toContainText('Nothing here yet. Once you trade');
      await expect(page.getByRole('region', { name: 'Pending', exact: true })).toContainText('No pending trades.');
    });

    test(`UX-034: proposal, acceptance and dispute with ${preference}, ${viewport.name}`, async ({ as, st }) => {
      const alice = await as('alice');
      const bob = await as('bob');
      for (const page of [alice, bob]) await page.setViewportSize({ width: viewport.width, height: viewport.height });
      const id = await preferredProposal(alice, st, preference, '0.05', 'Text preference acceptance and dispute');
      expect([st.tradePending, st.tradeDone, st.tradeReverse]).not.toContain(id);
      const tradePath = `/n/${st.slug}/trades/${id}`;
      await bob.goto(tradePath);
      await preferredAction(bob, preference, bob.getByRole('link', { name: 'Review and accept', exact: true }));
      await preferredAction(bob, preference, bob.getByRole('button', { name: 'Confirm and accept', exact: true }));
      await preferControls(bob, preference, [bob.locator('h1')]);
      await expect(bob.locator('#flash')).toContainText('Trade completed.');

      await alice.goto(tradePath);
      await preferredAction(alice, preference, alice.getByRole('link', { name: 'Raise a dispute', exact: true }));
      const statement = alice.getByLabel('What went wrong?', { exact: true });
      const review = alice.getByRole('button', { name: 'Review dispute', exact: true });
      await preferControls(alice, preference, [alice.locator('h1'), statement, review]);
      await statement.fill('The agreed delivery is still missing.');
      await review.click();
      await expect(alice.locator('h1')).toHaveText('Review your dispute');
      await preferredAction(alice, preference, alice.getByRole('button', { name: 'Open dispute', exact: true }));
      await expect(alice.locator('#flash')).toContainText('Dispute opened. The trade is marked Disputed. Your balance has not changed.');
      const home = alice.getByRole('link', { name: 'Home', exact: true });
      await preferredAction(alice, preference, home);
      const balance = alice.getByRole('region', { name: 'Your balance', exact: true });
      const history = alice.getByRole('link', { name: 'History', exact: true });
      await preferControls(alice, preference, [alice.locator('h1'), balance, history]);
      await expect(balance).toContainText('Available to spend');
      await history.click();
      const row = alice.locator('main li').filter({ has: alice.locator(`a[href="${tradePath}"]`) });
      await preferControls(alice, preference, [alice.locator('h1'), row]);
      await expect(row).toContainText('Disputed');
    });

    test(`UX-034: proposal and decline with ${preference}, ${viewport.name}`, async ({ as, st }) => {
      const alice = await as('alice');
      const bob = await as('bob');
      for (const page of [alice, bob]) await page.setViewportSize({ width: viewport.width, height: viewport.height });
      const id = await preferredProposal(alice, st, preference, '0.06', 'Text preference decline');
      expect(id).not.toBe(st.tradePending);
      const tradePath = `/n/${st.slug}/trades/${id}`;
      await bob.goto(tradePath);
      await preferredAction(bob, preference, bob.getByRole('link', { name: 'Decline', exact: true }));
      await expect(bob.locator('h1')).toHaveText('Decline this trade?');
      await preferredAction(bob, preference, bob.getByRole('button', { name: 'Decline this trade', exact: true }));
      await preferControls(bob, preference, [bob.locator('h1')]);
      await expect(bob.locator('#flash')).toContainText('Trade declined. Nothing was posted.');
      await expect(bob.getByText('Ledger reference', { exact: true })).toHaveCount(0);
      await alice.goto(`/n/${st.slug}/history`);
      const row = alice.locator('main li').filter({ has: alice.locator(`a[href="${tradePath}"]`) });
      await preferControls(alice, preference, [alice.locator('h1'), row]);
      await expect(row).toContainText('Declined');
    });

    test(`UX-034: dispute review and edit with ${preference}, ${viewport.name}`, async ({ as, st }) => {
      const page = await as('bob');
      await page.setViewportSize({ width: viewport.width, height: viewport.height });
      await page.goto(`/n/${st.slug}/trades/${st.tradeDone}/dispute`);
      const statement = page.getByLabel('What went wrong?', { exact: true });
      const review = page.getByRole('button', { name: 'Review dispute', exact: true });
      await preferControls(page, preference, [page.locator('h1'), statement, review]);
      const text = 'The agreed work has not been delivered.';
      await statement.fill(text);
      await review.click();
      await expect(page.locator('h1')).toHaveText('Review your dispute');
      await expect(page.locator('main')).toContainText(text);
      const open = page.getByRole('button', { name: 'Open dispute', exact: true });
      const edit = page.getByRole('button', { name: 'Go back and edit', exact: true });
      await preferControls(page, preference, [page.locator('h1'), open, edit]);
      await expect(open).toBeEnabled();
      // Stop before Open dispute; preview/edit does not mutate the seeded trade.
      await edit.click();
      await expect(statement).toHaveValue(text);
      await preferControls(page, preference, [page.locator('h1'), statement, review]);
      await expect(review).toBeEnabled();
    });
  }
}
