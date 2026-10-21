// Keyboard-only journeys. These tests never click, hover or use page.fill: every interaction is a key press
// (page.keyboard), so a control that cannot be reached or operated from the keyboard makes the test fail.
const { test, expect } = require('../fixtures');

/** Accessible-ish name of the focused element (aria-label, label text, text content, value, alt). */
async function focused(page) {
  return page.evaluate(() => {
    const el = document.activeElement;
    if (!el || el === document.body) return '';
    const label = el.labels && el.labels.length ? el.labels[0].innerText : '';
    const name = el.getAttribute('aria-label') || label || el.innerText || el.value || el.getAttribute('alt') || '';
    return name.replace(/\s+/g, ' ').trim();
  });
}

/** Presses Tab until the focused element's name matches; returns the Tab count. */
async function tabTo(page, re, max = 80) {
  for (let i = 1; i <= max; i++) {
    await page.keyboard.press('Tab');
    if (re.test(await focused(page))) return i;
  }
  throw new Error(`could not reach an element named ${re} by Tab within ${max} presses; last focus: "${await focused(page)}"`);
}

/** Every Tab stop on the page, in order, until focus leaves the document or wraps. */
async function tabOrder(page, max = 120) {
  const seen = [];
  for (let i = 0; i < max; i++) {
    await page.keyboard.press('Tab');
    const id = await page.evaluate(() => {
      const el = document.activeElement;
      if (!el || el === document.body) return null;
      const all = Array.from(document.querySelectorAll('*'));
      return { idx: all.indexOf(el), tag: el.tagName.toLowerCase() };
    });
    if (!id) break;
    if (seen.some((s) => s.idx === id.idx)) break; // wrapped around
    seen.push({ ...id, name: await focused(page) });
  }
  return seen;
}

async function typeText(page, text) {
  await page.keyboard.type(text, { delay: 5 });
}

async function keyboardLogin(page, login, password) {
  await page.goto('/login');
  await tabTo(page, /^Login$/);
  await typeText(page, login);
  await page.keyboard.press('Tab');
  expect(await focused(page)).toBe('Password');
  await typeText(page, password);
  await page.keyboard.press('Enter');
  await page.waitForURL((u) => u.pathname !== '/login');
}

test('skip link is the first Tab stop, becomes visible, and moves focus to main', async ({ page }) => {
  await page.goto('/login');
  await page.keyboard.press('Tab');
  expect(await focused(page)).toBe('Skip to main content');
  const box = await page.evaluate(() => {
    const r = document.activeElement.getBoundingClientRect();
    return { x: r.x, y: r.y, w: r.width, h: r.height, vw: innerWidth, vh: innerHeight };
  });
  expect(box.w).toBeGreaterThan(0);
  expect(box.h).toBeGreaterThan(0);
  expect(box.x).toBeGreaterThanOrEqual(0);
  expect(box.y).toBeGreaterThanOrEqual(0);
  expect(box.x + box.w).toBeLessThanOrEqual(box.vw);
  await page.keyboard.press('Enter');
  await expect(page).toHaveURL(/#main$/);
  expect(await page.evaluate(() => document.activeElement && document.activeElement.id)).toBe('main');
  // The next Tab lands inside the page content, skipping the header.
  await page.keyboard.press('Tab');
  expect(await focused(page)).toBe('Login');
});

test('login page tab order follows the visual and reading order', async ({ page }) => {
  await page.goto('/login');
  const order = (await tabOrder(page)).map((s) => s.name);
  const at = (re) => order.findIndex((n) => re.test(n));
  expect(order[0]).toBe('Skip to main content');
  const idx = [/^Skip/, /^Login$/, /^Password$/, /^Sign in$/, /Create an account/].map(at);
  expect(idx.every((i) => i >= 0), `all controls reachable, got ${JSON.stringify(order)}`).toBe(true);
  expect([...idx].sort((a, b) => a - b)).toEqual(idx);
});

test('no positive tabindex and a visible focus indicator on every Tab stop', async ({ as, st }) => {
  const page = await as('alice');
  const pages = ['/login', `/n/${st.slug}`, `/n/${st.slug}/trades/new`, `/n/${st.slug}/directory`, `/n/${st.slug}/history`];
  for (const p of pages) {
    await page.goto(p);
    const positive = await page.evaluate(() => Array.from(document.querySelectorAll('[tabindex]')).filter((e) => Number(e.getAttribute('tabindex')) > 0).length);
    expect(positive, `positive tabindex on ${p}`).toBe(0);
    const stops = await tabOrder(page);
    expect(stops.length, `tab stops on ${p}`).toBeGreaterThan(3);
    // Re-walk, checking the focus ring of each stop.
    await page.goto(p);
    for (let i = 0; i < stops.length; i++) {
      await page.keyboard.press('Tab');
      const ring = await page.evaluate(() => {
        const cs = getComputedStyle(document.activeElement);
        return { outline: cs.outlineStyle !== 'none' && parseFloat(cs.outlineWidth) >= 2, shadow: cs.boxShadow !== 'none' };
      });
      expect(ring.outline || ring.shadow, `focus indicator on "${await focused(page)}" at ${p}`).toBe(true);
    }
  }
});

test('sign in, propose a trade and accept it, using only the keyboard', async ({ browser, st }) => {
  const amount = '3.50';
  const ctxOpts = { baseURL: st.base, bypassCSP: true };

  // Alice proposes to Bob.
  const ac = await browser.newContext(ctxOpts);
  const a = await ac.newPage();
  await keyboardLogin(a, st.users.alice, st.password);
  await expect(a.locator('h1')).toBeVisible();
  expect(new URL(a.url()).pathname).toMatch(new RegExp(`^/n/${st.slug}/?$`));

  await tabTo(a, /^Propose a trade$/);
  await a.keyboard.press('Enter');
  await expect(a).toHaveURL(new RegExp(`/n/${st.slug}/trades/new`));

  await tabTo(a, /^Who is the trade with\?$/);
  await a.keyboard.type('Bob'); // type-ahead on the focused select
  expect(await a.evaluate(() => document.activeElement.selectedOptions[0].textContent)).toMatch(/Bob/);
  await tabTo(a, /^I pay them$/);
  expect(await a.evaluate(() => document.activeElement.checked)).toBe(true);
  await tabTo(a, /^Amount$/);
  await typeText(a, amount);
  await tabTo(a, /^What is it for/);
  await typeText(a, 'keyboard-only trade');
  await tabTo(a, /^Review trade$/);
  await a.keyboard.press('Enter');
  await expect(a.locator('h1')).toHaveText(/Review/i);
  await tabTo(a, /^Send trade$/);
  await a.keyboard.press('Enter');
  await expect(a.getByText('Trade sent to')).toBeVisible();
  const tradeId = new URL(a.url()).pathname.split('/').pop();
  await ac.close();

  // Bob finds it on his dashboard and accepts.
  const bc = await browser.newContext(ctxOpts);
  const b = await bc.newPage();
  await keyboardLogin(b, st.users.bob, st.password);
  expect(new URL(b.url()).pathname).toMatch(new RegExp(`^/n/${st.slug}/?$`));
  await tabTo(b, new RegExp(amount.replace('.', '\\.')));
  await b.keyboard.press('Enter');
  await expect(b).toHaveURL(new RegExp(`/trades/${tradeId}$`));
  await tabTo(b, /^Review and accept$/);
  await b.keyboard.press('Enter');
  await expect(b.locator('h1')).toBeVisible();
  await tabTo(b, /^Confirm and accept$/);
  await b.keyboard.press('Enter');
  await expect(b.getByText('Trade completed.')).toBeVisible();
  await bc.close();
});

test('validation errors are announced and reachable from the keyboard', async ({ page }) => {
  await page.goto('/login');
  await tabTo(page, /^Login$/);
  await typeText(page, 'nobody@example.org');
  await page.keyboard.press('Tab');
  await typeText(page, 'not the password');
  await page.keyboard.press('Enter');
  const summary = page.locator('#error-summary');
  await expect(summary).toBeVisible();
  await expect(summary).toHaveAttribute('role', 'alert');
  // The script moves focus to the summary, so the next Tab goes to the first error link.
  expect(await page.evaluate(() => document.activeElement.id)).toBe('error-summary');
  await page.keyboard.press('Tab');
  expect(await page.evaluate(() => document.activeElement.tagName)).toBe('A');
  await page.keyboard.press('Enter');
  expect(await page.evaluate(() => document.activeElement.id)).toBe('f-login');
});
