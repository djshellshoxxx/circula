// Keyboard-only helpers shared by the specs that must not click or fill.

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

/**
 * Fills a form with the keyboard only and activates the named button with Enter. fields is a list of
 * [labelRegExp, text] pairs in page order; text === true toggles a checkbox with Space. Resolves after the
 * resulting navigation.
 */
async function kbSubmit(page, fields, button) {
  for (const [re, text] of fields) {
    await tabTo(page, re);
    if (text === true) await page.keyboard.press('Space');
    else await page.keyboard.type(text, { delay: 3 });
  }
  await tabTo(page, button);
  await Promise.all([page.waitForNavigation(), page.keyboard.press('Enter')]);
}

/** Moves to a link or button by Tab and activates it with Enter. */
async function kbActivate(page, re) {
  await tabTo(page, re);
  await Promise.all([page.waitForNavigation(), page.keyboard.press('Enter')]);
}

module.exports = { focused, tabTo, kbSubmit, kbActivate };
