// UX-035 (WCAG 2.2 success criterion 2.5.8, Target Size (Minimum)) measured on every page of the shared seed, at
// 1280 px and on a 320 px touch layout. The rules (24 px minimum with the 2.5.8 exceptions, 44 px for the controls
// sized for touch) are in targets.js. axe's target-size rule (tag wcag22aa) runs on the same pages in a11y.spec.js;
// this is an independent measurement that also covers the touch-layout recommendation.
const path = require('path');
const { test, expect, STATE_DIR } = require('../fixtures');
const { PAGES } = require('../pages');
const { measure, targetSizeFailures } = require('../targets');

const LAYOUTS = [
  { name: '1280 px', viewport: { width: 1280, height: 900 } },
  { name: '320 px touch', viewport: { width: 320, height: 640 }, hasTouch: true, isMobile: true },
];

for (const layout of LAYOUTS) {
  test(`UX-035: target sizes on every page, ${layout.name}`, async ({ browser, st }) => {
    test.setTimeout(240_000);
    const contexts = {};
    const pageFor = async (who) => {
      const key = who || 'anonymous';
      if (!contexts[key]) {
        contexts[key] = await browser.newContext({
          baseURL: st.base, viewport: layout.viewport, hasTouch: !!layout.hasTouch, isMobile: !!layout.isMobile,
          storageState: who ? path.join(STATE_DIR, who + '.json') : undefined,
        });
      }
      return contexts[key].newPage();
    };
    const failures = [];
    let measured = 0;
    try {
      for (const [name, who, pathOf] of PAGES) {
        const page = await pageFor(who);
        await page.goto(pathOf(st));
        if (layout.hasTouch) expect(await page.evaluate(() => matchMedia('(pointer: coarse)').matches), 'touch layout').toBe(true);
        const r = await targetSizeFailures(page, !!layout.hasTouch);
        measured += r.count;
        failures.push(...r.failures.map((f) => `${name}: ${f}`));
        await page.close();
      }
    } finally {
      for (const c of Object.values(contexts)) await c.close();
    }
    expect(measured, 'targets measured').toBeGreaterThan(PAGES.length * 5);
    expect(failures, `targets that fail UX-035 at ${layout.name}`).toEqual([]);
  });
}

// Guard against a false green: crowded small targets and a small control on a touch layout are reported.
test('UX-035 guard: crowded 16 px links and a 30 px button are reported', async ({ browser, st }) => {
  const ctx = await browser.newContext({ baseURL: st.base, viewport: { width: 320, height: 640 }, hasTouch: true, isMobile: true });
  try {
    const page = await ctx.newPage();
    await page.goto('/login');
    await page.evaluate(() => {
      const box = document.createElement('div');
      for (const label of ['a', 'b']) {
        const a = document.createElement('a');
        a.href = '#';
        a.textContent = label;
        a.style.cssText = 'display:inline-block;width:16px;height:16px;margin:0;padding:0';
        box.appendChild(a);
      }
      const b = document.createElement('button');
      b.textContent = 'x';
      b.style.cssText = 'min-height:0;min-width:0;height:30px;width:30px;padding:0';
      box.appendChild(b);
      document.querySelector('main').appendChild(box);
    });
    const r = await page.evaluate(measure, true);
    expect(r.failures.filter((f) => /^a\b/.test(f)).length).toBe(2);
    expect(r.comfy.some((f) => f.startsWith('button') && f.includes('30x30'))).toBe(true);
  } finally {
    await ctx.close();
  }
});
