// UX-005 and QA-036: balance direction and trade status are not conveyed by colour alone. Each screen is shown three
// ways: in greyscale (Chromium's achromatopsia emulation), with forced colours (Windows high contrast style), and with
// every author colour removed (black text on white). In each one the visible text must still carry the meaning: the
// sign character and the word for every balance (+ In credit, − In debt, unsigned Level), the spelled-out direction
// for assistive technology, a sign on every signed amount and a status word on every trade row. The text must stay
// visible against its background, and the words on screen must be the same as in the normal presentation.
const { test, expect } = require('../fixtures');

const n = (st) => `/n/${st.slug}`;

// [name, who, path, balance states expected on the page]
const SCREENS = [
  ['dashboard, in debt', 'alice', (st) => n(st), ['In debt']],
  ['dashboard, in credit, with a trade waiting', 'bob', (st) => n(st), ['In credit']],
  ['dashboard, level', 'admin', (st) => n(st), ['Level']],
  ['history', 'alice', (st) => `${n(st)}/history`, []],
  ['trades', 'bob', (st) => `${n(st)}/trades`, []],
  ['trade detail, pending', 'bob', (st) => `${n(st)}/trades/${st.tradePending}`, []],
  ['accept review, balance before and after', 'bob', (st) => `${n(st)}/trades/${st.tradePending}/accept`, ['In credit']],
];

const MODES = {
  greyscale: async (page) => {
    const cdp = await page.context().newCDPSession(page);
    await cdp.send('Emulation.setEmulatedVisionDeficiency', { type: 'achromatopsia' });
  },
  'forced colours': async (page) => {
    await page.emulateMedia({ forcedColors: 'active' });
    expect(await page.evaluate(() => matchMedia('(forced-colors: active)').matches), 'forced colours applied').toBe(true);
  },
  'colours removed': async (page) => {
    await page.addStyleTag({
      content: '*,*::before,*::after{color:#000!important;background:#fff!important;border-color:#000!important;outline-color:#000!important}',
    });
  },
};

const BALANCE = /^([+−]?)(\d[\d,]*\.\d+) · (In credit|In debt|Level)$/;

/** What the page says, and how visible it is: read from the DOM and computed styles. */
async function readScreen(page) {
  return page.evaluate(() => {
    const parse = (s) => (s.match(/[\d.]+/g) || []).slice(0, 3).map(Number);
    const lum = ([r, g, b]) => [r, g, b].map((v) => {
      const c = v / 255;
      return c <= 0.04045 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4;
    }).reduce((a, c, i) => a + c * [0.2126, 0.7152, 0.0722][i], 0);
    const background = (el) => {
      for (let e = el; e; e = e.parentElement) {
        const bg = getComputedStyle(e).backgroundColor;
        const m = bg.match(/rgba?\(([^)]+)\)/);
        if (m && (m[1].split(',')[3] === undefined || Number(m[1].split(',')[3]) > 0)) return bg;
      }
      return 'rgb(255, 255, 255)';
    };
    const contrast = (el) => {
      const [a, b] = [lum(parse(getComputedStyle(el).color)), lum(parse(background(el)))].sort((x, y) => y - x);
      return Math.round(((a + 0.05) / (b + 0.05)) * 100) / 100;
    };
    const shown = (el) => {
      const r = el.getBoundingClientRect();
      const cs = getComputedStyle(el);
      return r.width > 0 && r.height > 0 && cs.visibility !== 'hidden' && Number(cs.opacity) > 0;
    };
    const balances = [...document.querySelectorAll('.bal')].map((b) => {
      const vis = b.querySelector('[aria-hidden="true"]');
      return {
        visible: vis.innerText.replace(/\s+/g, ' ').trim(),
        spoken: b.querySelector('.sr').textContent.trim(),
        shown: shown(vis),
        contrast: Math.min(...[...vis.querySelectorAll('.num, strong')].map(contrast)),
      };
    });
    const rows = [...document.querySelectorAll('ul.rows > li')].filter((li) => li.querySelector('.num')).map((li) => ({
      amount: li.querySelector('.num').innerText.trim(),
      status: li.querySelector('.tag') ? li.querySelector('.tag').innerText.trim() : '',
      shown: shown(li.querySelector('.num')) && (!li.querySelector('.tag') || shown(li.querySelector('.tag'))),
      contrast: Math.min(contrast(li.querySelector('.num')), li.querySelector('.tag') ? contrast(li.querySelector('.tag')) : 21),
    }));
    const tags = [...document.querySelectorAll('main .tag')].map((t) => ({ text: t.innerText.trim(), shown: shown(t), contrast: contrast(t) }));
    return { balances, rows, tags, text: document.querySelector('main').innerText.replace(/\s+/g, ' ').trim() };
  });
}

for (const [name, who, pathOf, states] of SCREENS) {
  test(`UX-005: ${name} is understandable without colour`, async ({ as, st }) => {
    const page = await as(who);
    await page.goto(pathOf(st));
    const normal = await readScreen(page);
    expect(normal.balances.map((b) => b.visible.match(BALANCE)?.[3]), `${name}: balance states`).toEqual(expect.arrayContaining(states));
    if (name.startsWith('history') || name.startsWith('trades')) expect(normal.rows.length, `${name}: trade rows`).toBeGreaterThan(0);

    for (const [mode, apply] of Object.entries(MODES)) {
      const p = await as(who); // a fresh page, so no emulation carries over from the previous mode
      await p.goto(pathOf(st));
      await apply(p);
      const s = await readScreen(p);
      const where = `${name}, ${mode}`;
      expect(s.text, `${where}: the same words as in colour`).toBe(normal.text);
      for (const b of s.balances) {
        const m = b.visible.match(BALANCE);
        expect(m, `${where}: balance "${b.visible}" shows a sign and a word`).not.toBeNull();
        const [, sign, amount, state] = m;
        if (state === 'In credit') expect(sign, `${where}: ${b.visible}`).toBe('+');
        if (state === 'In debt') expect(sign, `${where}: ${b.visible}`).toBe('−');
        if (state === 'Level') expect([sign, Number(amount)], `${where}: ${b.visible}`).toEqual(['', 0]);
        // assistive technology gets the direction in words, not the minus sign
        expect(b.spoken, `${where}: spoken balance`).toMatch(state === 'Level' ? /^Level, zero / : new RegExp(`^${state} by ${amount} `));
        expect(b.shown, `${where}: balance is on screen`).toBe(true);
        expect(b.contrast, `${where}: balance text contrast`).toBeGreaterThanOrEqual(4.5);
      }
      for (const r of s.rows) {
        expect(r.amount, `${where}: signed amount`).toMatch(/^[+−]\d/);
        expect(r.status, `${where}: status word for ${r.amount}`).toMatch(/^(Completed|Waiting for .+ to respond|Declined|Cancelled|Expired|Disputed|Reversed)$/);
        expect(r.shown, `${where}: amount and status are on screen`).toBe(true);
        expect(r.contrast, `${where}: amount and status contrast`).toBeGreaterThanOrEqual(4.5);
      }
      for (const t of s.tags) {
        expect(t.text, `${where}: status tag has words`).not.toBe('');
        expect(t.shown && t.contrast >= 4.5, `${where}: status tag "${t.text}" visible (contrast ${t.contrast})`).toBe(true);
      }
    }
  });
}

// Guard against a false green: a balance that only a colour distinguishes is reported.
test('UX-005 guard: a colour-only balance fails the sign-and-word check', async ({ as, st }) => {
  const page = await as('alice');
  await page.goto(n(st));
  await page.evaluate(() => {
    const vis = document.querySelector('.bal [aria-hidden="true"]');
    const num = vis.querySelector('.num');
    num.textContent = '14.00';
    num.style.color = 'rgb(155, 28, 28)';
    vis.replaceChildren(num); // only the number, its direction left to the colour
  });
  const s = await readScreen(page);
  expect(s.balances[0].visible).not.toMatch(BALANCE);
});
