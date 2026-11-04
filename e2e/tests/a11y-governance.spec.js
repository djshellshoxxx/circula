// Axe (WCAG 2.2 AA rule set, light and dark colour schemes) of the page templates and states that need network
// settings the shared seed does not have: membership by approval (request form, its error, the waiting page and the
// approval step), published terms (administrator form, acceptance form and their errors), the leave form of a member
// who is free to leave, and the administrator's consistency check result and adjustment review step.
//
// It runs against its own instance and database (startInstance in global-setup.js), seeded through the web forms with
// the shared helpers, so the shared seed used by the other specs is not changed. Each check also measures the UX-035
// target sizes (targets.js). The tests are ordered and serial.
const { test, expect } = require('@playwright/test');
const { expectNoAxeViolations } = require('../fixtures');
const { expectTargetSizes } = require('../targets');
const gs = require('../global-setup');

const { register, login, submit, SETUP_SECRET, SLUG } = gs.helpers;
const USERS = { admin: 'gadmin@example.org', pat: 'gpat@example.org' };
const n = `/n/${SLUG}`;

test.describe.configure({ mode: 'serial' });

let inst; // the instance
const signedIn = {}; // storage state per user, captured at sign-in

test.beforeAll(async ({ browser }) => {
  test.setTimeout(120_000);
  inst = await gs.startInstance();
  const ctx = await browser.newContext({ baseURL: inst.base });
  const p = await ctx.newPage();
  await register(p, USERS.admin);
  await login(p, USERS.admin);
  await p.goto('/setup');
  await p.getByLabel('Setup secret').fill(SETUP_SECRET);
  await p.getByLabel('Network name').fill('Approval Circle');
  await p.getByLabel('Web address name').fill(SLUG);
  await p.getByLabel('Your display name in this network').fill('Gail Admin');
  await p.getByLabel('Credit unit name').fill('credit');
  await p.getByLabel('Default credit limit per member').fill('100.00');
  await p.getByLabel('Sign-up grant').fill('0.00');
  await p.getByLabel('How people join').selectOption('approval');
  await submit(p, 'Create network');
  if (!new URL(p.url()).pathname.startsWith(n)) throw new Error('network not created, at ' + p.url());
  signedIn.admin = await ctx.storageState();
  await ctx.close();
});
test.afterAll(async () => {
  if (inst) await inst.stop();
});

async function as(browser, who) {
  const ctx = await browser.newContext({ baseURL: inst.base, bypassCSP: true, storageState: signedIn[who] });
  return { ctx, page: await ctx.newPage() };
}

// axe in both colour schemes, then the UX-035 target sizes at this width and at 320 px (targets.js).
async function checkPage(page, label) {
  await expect(page.locator('h1'), `${label}: one h1`).toHaveCount(1);
  await expectNoAxeViolations(page, label);
  await expectTargetSizes(page, label);
}

test('axe: administrator terms, error state and a published version', async ({ browser }) => {
  const { ctx, page } = await as(browser, 'admin');
  await page.goto(`${n}/admin/terms`);
  await submit(page, 'Publish new version'); // empty text: error state
  await expect(page.locator('#error-summary')).toBeVisible();
  await checkPage(page, 'admin terms error state');
  await page.getByLabel('Terms text').fill('Trade fairly and describe what you offer honestly.\n\nAsk an administrator when in doubt.');
  await submit(page, 'Publish new version');
  await expect(page.locator('#flash')).toContainText('Published version 1');
  await checkPage(page, 'admin terms (version 1 published)');
  await ctx.close();
});

test('axe: network terms acceptance form and its error state', async ({ browser }) => {
  const { ctx, page } = await as(browser, 'admin');
  await page.goto(`${n}/terms`);
  await expect(page.getByRole('checkbox', { name: /accept version 1/ })).toBeVisible();
  await checkPage(page, 'network terms (acceptance needed)');
  await submit(page, 'Accept these terms'); // box not ticked: error state
  await expect(page.locator('#error-summary')).toBeVisible();
  await checkPage(page, 'network terms acceptance error state');
  await page.getByRole('checkbox', { name: /accept version 1/ }).check();
  await submit(page, 'Accept these terms');
  await expect(page.locator('#flash')).toContainText('You accepted version 1');
  await ctx.close();
});

test('axe: request to join by approval, its error state, then the waiting page', async ({ browser }) => {
  const ctx = await browser.newContext({ baseURL: inst.base, bypassCSP: true });
  const page = await ctx.newPage();
  await register(page, USERS.pat);
  await login(page, USERS.pat);
  await page.goto('/'); // no membership yet: the network list, not a redirect to a dashboard
  await expect(page.locator('h1')).toHaveText('Your networks');
  await checkPage(page, 'networks list (no network yet)');
  await page.goto(`${n}/join`);
  await expect(page.getByRole('button', { name: 'Request to join' })).toBeVisible();
  await checkPage(page, 'join a network (approval, with terms)');
  await submit(page, 'Request to join'); // no name, terms not accepted: error state
  await expect(page.locator('#error-summary')).toBeVisible();
  await checkPage(page, 'join request error state');
  await page.getByLabel('Your display name').fill('Pat Member');
  await page.getByRole('checkbox', { name: 'I have read and accept these terms' }).check();
  await submit(page, 'Request to join');
  await expect(page.locator('h1')).toHaveText('Waiting for approval');
  await checkPage(page, 'waiting for approval');
  signedIn.pat = await ctx.storageState();
  await ctx.close();
});

test('axe: administrator approves the request', async ({ browser }) => {
  const { ctx, page } = await as(browser, 'admin');
  await page.goto(`${n}/admin/members?status=pending`);
  await page.getByRole('link', { name: 'Approve Pat Member' }).click();
  await expect(page.getByRole('button', { name: 'Approve', exact: true })).toBeVisible();
  await checkPage(page, 'admin member action (approve)');
  await submit(page, 'Approve');
  await expect(page.locator('#flash')).toBeVisible();
  await ctx.close();
});

test('axe: leave form of a member who can leave, and its error state', async ({ browser }) => {
  const { ctx, page } = await as(browser, 'pat');
  await page.goto(`${n}/leave`);
  await expect(page.getByRole('button', { name: 'Leave this network' })).toBeVisible();
  await checkPage(page, 'leave the network (form)');
  await submit(page, 'Leave this network'); // not confirmed: error state, nothing changes
  await expect(page.locator('#error-summary')).toBeVisible();
  await checkPage(page, 'leave the network error state');
  await ctx.close();
});

test('axe: consistency check result and adjustment review step', async ({ browser }) => {
  const { ctx, page } = await as(browser, 'admin');
  await page.goto(`${n}/admin/audit`);
  await submit(page, 'Run consistency check');
  await expect(page.locator('h1')).toHaveText('Consistency check result');
  await expect(page.getByText('Check passed: no problems found.')).toBeVisible();
  await checkPage(page, 'consistency check result');

  await page.goto(`${n}/admin/adjust`);
  await page.getByLabel('Member', { exact: true }).selectOption({ label: 'Pat Member' });
  await page.getByLabel('Amount').fill('1.00');
  await page.getByLabel('Reason').fill('Welcome credit for the first market stall.');
  await submit(page, 'Review adjustment');
  await expect(page.getByRole('button', { name: 'Post adjustment' })).toBeVisible();
  await checkPage(page, 'adjustment review step'); // not posted
  await ctx.close();
});
