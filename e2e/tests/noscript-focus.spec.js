// QA-034 / UX-040: visible keyboard focus on the named core-flow entry screens.
// GET-only: no control is activated and no seeded transaction is changed.
const path = require('path');
const { test, expect, STATE_DIR } = require('../fixtures');

const VIEWPORTS = [
  { name: 'portrait', width: 320, height: 640 },
  { name: 'landscape', width: 640, height: 320 },
];
const PAGES = [
  ['registration', null, () => '/register'],
  ['invitation (signed out)', null, (st) => st.unusedInvite],
  ['invitation (signed in)', 'alice', (st) => st.unusedInvite],
  ['proposal', 'alice', (st) => `/n/${st.slug}/trades/new?to=${st.bobId}`],
  ['acceptance', 'bob', (st) => `/n/${st.slug}/trades/${st.tradePending}/accept`],
  ['decline', 'bob', (st) => `/n/${st.slug}/trades/${st.tradePending}/decline`],
  ['balance', 'alice', (st) => `/n/${st.slug}`],
  ['history', 'alice', (st) => `/n/${st.slug}/history`],
  ['dispute raise', 'bob', (st) => `/n/${st.slug}/trades/${st.tradeDone}/dispute`],
];

for (const viewport of VIEWPORTS) {
  for (const [name, who, pathOf] of PAGES) {
    test(`QA-034/UX-040: no-script focus, ${name}, ${viewport.name}`, async ({ browser, st }) => {
      const options = {
        baseURL: st.base, javaScriptEnabled: false,
        viewport: { width: viewport.width, height: viewport.height },
      };
      if (who) options.storageState = path.join(STATE_DIR, who + '.json');
      const context = await browser.newContext(options);
      try {
        const page = await context.newPage();
        const response = await page.goto(pathOf(st));
        expect(response).not.toBeNull();
        expect(response.status()).toBe(200);
        const positive = await page.locator('[tabindex]').evaluateAll((elements) => elements.filter((el) => el.tabIndex > 0).length);
        expect(positive, 'no positive tabindex').toBe(0);
        const seen = new Set();
        let ended = false;
        for (let i = 0; i < 120; i++) {
          await page.keyboard.press('Tab');
          const stop = await page.evaluate(() => {
            const el = document.activeElement;
            if (!el || el === document.body) return null;
            const cs = getComputedStyle(el);
            const rect = el.getBoundingClientRect();
            const left = Math.max(0, rect.left), right = Math.min(innerWidth, rect.right);
            const top = Math.max(0, rect.top), bottom = Math.min(innerHeight, rect.bottom);
            let unoccluded = false;
            if (right > left && bottom > top) {
              for (const x of [0.1, 0.5, 0.9]) for (const y of [0.1, 0.5, 0.9]) {
                const hit = document.elementFromPoint(left + (right - left) * x, top + (bottom - top) * y);
                if (hit && (hit === el || el.contains(hit))) unoccluded = true;
              }
            }
            return {
              index: Array.from(document.querySelectorAll('*')).indexOf(el),
              name: (el.getAttribute('aria-label') || el.innerText || el.tagName).replace(/\s+/g, ' ').trim(),
              ring: (cs.outlineStyle !== 'none' && parseFloat(cs.outlineWidth) >= 2) || cs.boxShadow !== 'none',
              visible: cs.visibility === 'visible' && parseFloat(cs.opacity) > 0 && right > left && bottom > top,
              unoccluded,
            };
          });
          if (!stop || seen.has(stop.index)) { ended = true; break; }
          seen.add(stop.index);
          expect(stop.visible, `${stop.name}: focused control intersects viewport`).toBe(true);
          expect(stop.unoccluded, `${stop.name}: a sampled visible focus point is not covered`).toBe(true);
          expect(stop.ring, `${stop.name}: visible focus indicator`).toBe(true);
        }
        expect(ended, 'tab traversal ends or wraps within 120 stops').toBe(true);
        expect(seen.size, 'core-screen controls are keyboard reachable').toBeGreaterThan(3);
      } finally {
        await context.close();
      }
    });
  }
}
