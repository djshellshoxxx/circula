// UX-038 / UX-055: keyboard-only administration and listing journeys in the approval-mode instance (apply.js).
// Each journey starts on Home and then uses only key presses: Tab to move, typing to fill, Space and arrow keys
// for choices, Enter to follow links and submit. Nothing is clicked, filled or opened by URL after Home, so a
// control that the keyboard cannot reach or operate fails the test. The shared demo network is not touched;
// Gail's request stays pending for the page checks in other specs.
const { test, expect } = require('../apply');
const { tabTo, kbSubmit, kbActivate } = require('../kbd');

const home = (st) => `/n/${st.apply.slug}`;

async function startAtHome(page, st) {
  await page.goto(home(st));
  await expect(page.locator('h1')).toHaveText('Home');
}

/** Replaces the text of the focused field with the keyboard. */
async function retype(page, text) {
  await page.keyboard.press('ControlOrMeta+A');
  await page.keyboard.press('Backspace');
  await page.keyboard.type(text, { delay: 3 });
}

test('UX-038: approve a request to join using only the keyboard', async ({ asApply, st }) => {
  const page = await asApply('erin');
  await startAtHome(page, st);
  await kbActivate(page, /^Network admin$/);
  await expect(page.locator('h1')).toHaveText('Network admin');
  await expect(page.locator('main')).toContainText('membership requests are waiting');
  await kbActivate(page, /^Review requests$/);
  await expect(page).toHaveURL((url) => url.pathname === `${home(st)}/admin/members` && url.searchParams.get('status') === 'pending');
  const finn = page.locator('tbody tr').filter({ hasText: 'Finn Applicant' });
  await expect(finn).toContainText('pending');
  await kbActivate(page, /^Approve Finn Applicant$/);
  await expect(page.locator('h1')).toHaveText('Approve: Finn Applicant?');
  await kbActivate(page, /^Approve$/);
  await expect(page.locator('#flash')).toContainText('Approve: done for Finn Applicant.');
  await expect(page.locator('tbody tr').filter({ hasText: 'Finn Applicant' })).toContainText('active');
  await expect(page.locator('tbody tr').filter({ hasText: 'Gail Applicant' })).toContainText('pending');
});

test('UX-038: create and then edit a listing using only the keyboard', async ({ asApply, st }) => {
  const page = await asApply('hana');
  await startAtHome(page, st);
  await kbActivate(page, /^Listings$/);
  await kbActivate(page, /^New listing$/);
  await expect(page.locator('h1')).toBeVisible();

  // The kind is a radio group: Tab reaches the checked option, an arrow key moves the choice.
  await tabTo(page, /^Offer: something I provide$/);
  await page.keyboard.press('ArrowDown');
  await expect(page.getByLabel('Request: something I want', { exact: true })).toBeChecked();
  await kbSubmit(page, [[/^Title$/, 'Keyboard-made shelf'], [/^Description$/, 'Wanted: a small wall shelf.'], [/^Price \(optional\)$/, '3.00']], /^Publish listing$/);
  await expect(page.locator('#flash')).toContainText('Listing published.');
  const created = page.locator('main li').filter({ hasText: 'Keyboard-made shelf' });
  await expect(created).toContainText('Request:');

  await kbActivate(page, /^Edit Keyboard-made shelf$/);
  await tabTo(page, /^Title$/);
  await retype(page, 'Keyboard-edited shelf');
  await tabTo(page, /^Status/);
  await page.keyboard.type('Paused'); // type-ahead on the focused select
  await expect(page.getByLabel('Status', { exact: true })).toHaveValue('paused');
  await kbActivate(page, /^Save$/);
  await expect(page.locator('#flash')).toContainText('Listing saved.');
  const edited = page.locator('main li').filter({ hasText: 'Keyboard-edited shelf' });
  await expect(edited).toContainText('paused');
  await expect(page.locator('main li').filter({ hasText: 'Keyboard-made shelf' })).toHaveCount(0);
});

test('UX-038: change the policy using only the keyboard', async ({ asApply, st }) => {
  const page = await asApply('erin');
  await startAtHome(page, st);
  await kbActivate(page, /^Network admin$/);
  await kbActivate(page, /^Policy$/);
  await expect(page.locator('h1')).toHaveText('Network policy');
  const window = page.getByLabel('Dispute window (days)', { exact: true });
  const before = Number(await window.inputValue());
  const after = String(before + 15); // a longer window is not adverse, so it applies at once
  await tabTo(page, /^Dispute window \(days\)$/);
  await retype(page, after);
  await tabTo(page, /^Reason for the change$/);
  await page.keyboard.type('Keyboard-only policy test: longer dispute window.', { delay: 3 });
  await kbActivate(page, /^Save new version$/);
  await expect(page.locator('#flash')).toContainText('Policy saved as a new version.');
  await expect(window).toHaveValue(after);
  await kbActivate(page, /^change log$/);
  await expect(page.locator('h1')).toHaveText('Network rules');
  await expect(page.locator('main')).toContainText(`Dispute window (days): ${before} to ${after}`);
  await expect(page.locator('main')).toContainText('Reason: Keyboard-only policy test: longer dispute window.');
});
