// QA-034 / UX-040: visible keyboard focus on the named core-flow entry screens.
// UX-039: focus order follows reading order. Each Tab stop comes later in the DOM than the one before
// (no positive tabindex), no CSS reorders a stop or its containers (order, reversed flex, grid placement),
// and each stop starts on the same line to the right of the one before or on a later line.
// GET-only: no control is activated and no seeded transaction is changed.
const { test, expect, session } = require('../apply');

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
  ['sign-in', null, () => '/login'],
  ['account security', 'alice', () => '/account/security'],
  ['network rules', 'alice', (st) => `/n/${st.slug}/policy`],
  ['new listing', 'alice', (st) => `/n/${st.slug}/listings/new`],
  ['edit own listing', 'alice', (st) => st.listingEdit],
  ['admin members', 'admin', (st) => `/n/${st.slug}/admin/members`],
  ['admin policy', 'admin', (st) => `/n/${st.slug}/admin/policy`],
  ['admin approve a request', 'apply:erin', (st) => `/n/${st.apply.slug}/admin/members/${st.apply.gailId}/approve`],
  ['network terms (to accept)', 'apply:hana', (st) => `/n/${st.apply.slug}/terms`],
];

for (const viewport of VIEWPORTS) {
  for (const [name, who, pathOf] of PAGES) {
    test(`QA-034/UX-039/UX-040: no-script focus, ${name}, ${viewport.name}`, async ({ browser, st }) => {
      const context = await browser.newContext({
        ...session(st, who), javaScriptEnabled: false,
        viewport: { width: viewport.width, height: viewport.height },
      });
      try {
        const page = await context.newPage();
        const response = await page.goto(pathOf(st));
        expect(response).not.toBeNull();
        expect(response.status()).toBe(200);
        const positive = await page.locator('[tabindex]').evaluateAll((elements) => elements.filter((el) => el.tabIndex > 0).length);
        expect(positive, 'no positive tabindex').toBe(0);
        const seen = new Set();
        let ended = false;
        let previous = null;
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
            // CSS that would show a stop somewhere other than its DOM position.
            let reordered = '';
            for (let node = el; node && node !== document.body && !reordered; node = node.parentElement) {
              const ns = getComputedStyle(node);
              const parent = node.parentElement ? getComputedStyle(node.parentElement) : null;
              if (ns.order !== '0') reordered = `order ${ns.order} on ${node.tagName}`;
              else if (parent && /flex/.test(parent.display) && /reverse/.test(parent.flexDirection + ' ' + parent.flexWrap)) reordered = `reversed flex around ${node.tagName}`;
              else if (parent && /grid/.test(parent.display) && (ns.gridRowStart !== 'auto' || ns.gridColumnStart !== 'auto' || /dense/.test(parent.gridAutoFlow))) reordered = `grid placement of ${node.tagName}`;
            }
            const first = el.getClientRects()[0] || rect;
            return {
              index: Array.from(document.querySelectorAll('*')).indexOf(el),
              name: (el.getAttribute('aria-label') || el.innerText || el.tagName).replace(/\s+/g, ' ').trim(),
              reordered,
              // Where the stop starts in the page (its first line box), independent of scrolling.
              start: { top: first.top + scrollY, bottom: first.bottom + scrollY, left: first.left + scrollX },
              ring: (cs.outlineStyle !== 'none' && parseFloat(cs.outlineWidth) >= 2) || cs.boxShadow !== 'none',
              visible: cs.visibility === 'visible' && parseFloat(cs.opacity) > 0 && right > left && bottom > top,
              unoccluded,
            };
          });
          if (!stop || seen.has(stop.index)) { ended = true; break; }
          seen.add(stop.index);
          expect(stop.reordered, `${stop.name}: no CSS moves this stop away from its DOM position`).toBe('');
          if (previous) {
            expect(stop.index, `${stop.name}: Tab moves forward in DOM (reading) order after ${previous.name}`).toBeGreaterThan(previous.index);
            const a = previous.start, b = stop.start;
            const overlap = Math.min(a.bottom, b.bottom) - Math.max(a.top, b.top);
            const sameLine = overlap > Math.min(a.bottom - a.top, b.bottom - b.top) / 2;
            if (sameLine) {
              expect(b.left, `${stop.name}: on the line of ${previous.name}, it starts to its right`).toBeGreaterThanOrEqual(a.left - 1);
            } else {
              expect(b.top, `${stop.name}: on a later line than ${previous.name}`).toBeGreaterThanOrEqual(a.top - 1);
            }
          }
          previous = stop;
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
