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
