// Session lifecycle against the actual server, with separate browser cookies and no page JavaScript.
// This instance is isolated from the shared accessibility seed because revocation changes authentication.
const { test, expect } = require('@playwright/test');
const { startInstance, helpers } = require('../global-setup');
const { kbSubmit } = require('../kbd');

test('sessions: no-JS pagination, owner isolation, confirmation and selective sign-out', async ({ browser }) => {
  test.setTimeout(180_000);
  const instance = await startInstance();
  const contexts = [];
  const open = async () => {
    const context = await browser.newContext({ baseURL: instance.base, javaScriptEnabled: false });
    contexts.push(context);
    return { context, page: await context.newPage() };
  };
  const sessionLinks = (page) => page.locator('main ul.rows a[href$="/revoke"]');
  try {
    const current = await open();
    await helpers.register(current.page, 'sessions-owner@example.org');
    await helpers.login(current.page, 'sessions-owner@example.org');

    const other = await open();
    await helpers.login(other.page, 'sessions-owner@example.org');
    await other.page.goto('/account/sessions');
    const otherPath = await other.page.getByRole('link', { name: 'Sign out here', exact: true }).getAttribute('href');
    expect(otherPath).toMatch(/^\/account\/sessions\/[0-9a-f]{64}\/revoke$/);

    const foreign = await open();
    await helpers.register(foreign.page, 'sessions-foreign@example.org');
    await helpers.login(foreign.page, 'sessions-foreign@example.org');
    await foreign.page.goto('/account/sessions');
    const foreignPath = await foreign.page.getByRole('link', { name: 'Sign out here', exact: true }).getAttribute('href');
    expect((await current.context.request.get(foreignPath)).status()).toBe(404);

    // Repeated successful logins retain distinct sessions. Keep the separate "other" browser's cookie.
    const seed = await open();
    for (let i = 0; i < 28; i++) await helpers.login(seed.page, 'sessions-owner@example.org');

    await current.page.goto('/account/sessions');
    await expect(sessionLinks(current.page)).toHaveCount(25);
    const first = await sessionLinks(current.page).evaluateAll((links) => links.map((link) => link.getAttribute('href')));
    await current.page.getByRole('link', { name: 'Next page', exact: true }).click();
    await expect(sessionLinks(current.page)).toHaveCount(5);
    await expect(current.page.getByRole('heading', { name: 'This session', exact: true })).toBeVisible();
    await current.page.getByRole('link', { name: 'Previous page', exact: true }).click();
    await expect(sessionLinks(current.page)).toHaveCount(25);
    expect(await sessionLinks(current.page).evaluateAll((links) => links.map((link) => link.getAttribute('href')))).toEqual(first);

    // Direct confirmation reaches an older session beyond page one, without ending it on GET.
    await current.page.goto(otherPath);
    await expect(current.page.getByText('You will stay signed in here.', { exact: false })).toBeVisible();
    expect((await other.context.request.get('/api/v1/me/sessions')).status()).toBe(200);
    expect((await current.context.request.post(otherPath, {
      form: { confirm: 'yes', csrf: 'invalid' }, maxRedirects: 0,
    })).status()).toBe(403);
    await current.page.getByRole('button', { name: 'End session', exact: true }).click();
    await expect(current.page.locator('#error-summary')).toBeVisible();
    await kbSubmit(current.page, [[/^I want to end this session\./, true]], /^End session$/);
    await expect(current.page).toHaveURL(instance.base + '/account/sessions');
    await other.page.goto('/account/sessions');
    await expect(other.page).toHaveURL(/\/login(?:\?|$)/);

    await current.page.getByRole('link', { name: 'Sign out other sessions', exact: true }).click();
    await expect(current.page.getByRole('heading', { name: 'Sign out other sessions?', exact: true })).toBeVisible();
    await current.page.getByRole('button', { name: 'Sign out other sessions', exact: true }).click();
    await expect(current.page.locator('#error-summary')).toBeVisible();
    await kbSubmit(current.page, [[/^I want to end all my other sessions\./, true]], /^Sign out other sessions$/);
    await expect(current.page).toHaveURL(instance.base + '/account/sessions');
    await expect(sessionLinks(current.page)).toHaveCount(1);
    await expect(current.page.getByRole('heading', { name: 'This session', exact: true })).toBeVisible();
    expect((await seed.context.request.get('/api/v1/me/sessions')).status()).toBe(401);
    expect((await foreign.context.request.get('/api/v1/me/sessions')).status()).toBe(200);

    await current.page.getByRole('link', { name: 'Sign out here', exact: true }).click();
    await kbSubmit(current.page, [[/^I want to end this session\./, true]], /^End session$/);
    await expect(current.page).toHaveURL(instance.base + '/login');
    expect((await current.context.request.get('/api/v1/me/sessions')).status()).toBe(401);
  } finally {
    for (const context of contexts) await context.close();
    await instance.stop();
  }
});
