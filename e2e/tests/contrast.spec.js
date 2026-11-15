// UX-057 and QA-036: contrast of every colour token of the stylesheet, in the light and in the dark scheme.
//
// The design tokens are the custom properties on :root in internal/web/static/style.css; the dark scheme redefines
// them under (prefers-color-scheme: dark). The browser resolves every token in each scheme, and the WCAG 2.x contrast
// ratio is computed here for each foreground and background pair the stylesheet combines: at least 4.5:1 for text
// (1.4.3; large text needs only 3:1, so 4.5 covers it) and 3:1 for UI component boundaries, state indicators and the
// focus indicator against each surface they sit on (1.4.11). The state colours (error, success, notice) are among
// them. axe checks rendered text in both schemes on every page (a11y.spec.js); this matrix also covers non-text
// colours and tokens that no seeded page happens to show.
const { test, expect, useScheme, SCHEMES } = require('../fixtures');

const TEXT = 4.5;
const UI = 3;

// [foreground token, background token, minimum ratio, where the stylesheet combines them]
const PAIRS = [
  ['fg', 'bg', TEXT, 'body text, headings, form field text'],
  ['muted', 'bg', TEXT, 'hints and footer text'],
  ['accent', 'bg', TEXT, 'links, secondary and link-style button text'],
  ['accent-fg', 'accent', TEXT, 'primary button and skip link text'],
  ['bad', 'bg', TEXT, 'field error messages'],
  ['bg', 'bad', TEXT, 'danger button text'],
  ['fg', 'badbg', TEXT, 'error summary text'],
  ['bad', 'badbg', TEXT, 'error summary heading'],
  ['accent', 'badbg', TEXT, 'error summary links to the fields'],
  ['fg', 'okbg', TEXT, 'success message text'],
  ['fg', 'note', TEXT, 'notice and session warning text'],
  ['accent', 'note', TEXT, 'links and link-style buttons in notices'],

  ['line', 'bg', UI, 'borders of form fields, fieldsets, tables, cards, tags and the meter'],
  ['accent', 'bg', UI, 'primary button fill, secondary button border'],
  ['accent', 'note', UI, 'button fill in the session warning'],
  ['bad', 'bg', UI, 'error border of a field and of the error summary, danger button fill'],
  ['ok', 'bg', UI, 'success message border'],
  ['focus', 'bg', UI, 'focus indicator on the page'],
  ['focus', 'note', UI, 'focus indicator inside a notice'],
  ['focus', 'badbg', UI, 'focus indicator inside the error summary'],
];

function channel(v) {
  const c = v / 255;
  return c <= 0.04045 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4;
}

/** Relative luminance (WCAG 2.x) of a computed colour "rgb(r, g, b)". */
function luminance(rgb) {
  const [r, g, b] = rgb;
  return 0.2126 * channel(r) + 0.7152 * channel(g) + 0.0722 * channel(b);
}

function contrast(a, b) {
  const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x);
  return (hi + 0.05) / (lo + 0.05);
}

function parseRgb(s) {
  const m = s.match(/^rgba?\((\d+), (\d+), (\d+)(?:, ([\d.]+))?\)$/);
  if (!m) throw new Error(`not an sRGB computed colour: ${s}`);
  return { rgb: [Number(m[1]), Number(m[2]), Number(m[3])], alpha: m[4] === undefined ? 1 : Number(m[4]) };
}

/**
 * Reads the stylesheet through the CSSOM: the token names declared on :root outside and inside the dark-scheme
 * media rule, every token's computed colour in the current scheme, and declarations outside the token blocks that
 * use a colour literal (which would escape this check).
 */
async function readTokens(page) {
  return page.evaluate(() => {
    const base = new Set();
    const dark = new Set();
    const literals = [];
    const keywords = new Set(['transparent', 'currentcolor', 'inherit', 'initial', 'unset', 'revert', 'revert-layer']);
    const fn = /#[0-9a-f]{3,8}\b|\b(?:rgba?|hsla?|hwb|lab|lch|oklab|oklch|color)\(/i;
    const walk = (rules, inDark) => {
      for (const r of rules) {
        const isDark = inDark || (r.media && /prefers-color-scheme:\s*dark/.test(r.media.mediaText));
        if (r.style) {
          for (const prop of r.style) {
            const value = r.style.getPropertyValue(prop);
            if (prop.startsWith('--')) {
              if (r.selectorText === ':root') (isDark ? dark : base).add(prop);
              continue;
            }
            const words = value.replace(/var\([^)]*\)/g, ' ').replace(/"[^"]*"/g, ' ').split(/[\s,/]+/).filter(Boolean);
            // named colours are bare identifiers; colour functions and hex values are caught by fn
            const named = words.filter((w) => /^[a-z]+$/i.test(w) && !keywords.has(w.toLowerCase()) && CSS.supports('color', w));
            if (fn.test(value) || named.length) literals.push(`${r.selectorText} { ${prop}: ${value} }`);
          }
        }
        if (r.cssRules) walk(r.cssRules, isDark);
      }
    };
    for (const sheet of document.styleSheets) walk(sheet.cssRules, false);
    const probe = document.createElement('span');
    document.body.appendChild(probe);
    const values = {};
    const notColours = [];
    const root = getComputedStyle(document.documentElement);
    for (const name of base) {
      // All tokens are colours; one that is not a valid colour would silently resolve to the inherited colour.
      if (!CSS.supports('color', root.getPropertyValue(name).trim())) notColours.push(name);
      probe.style.color = `var(${name})`;
      values[name.slice(2)] = getComputedStyle(probe).color;
    }
    probe.remove();
    return { base: [...base].map((x) => x.slice(2)).sort(), dark: [...dark].map((x) => x.slice(2)).sort(), values, literals, notColours };
  });
}

test('every colour is a token, every token has a dark value and is in the contrast matrix', async ({ page }) => {
  await page.goto('/login');
  const t = await readTokens(page);
  expect(t.base.length, 'tokens on :root').toBeGreaterThan(5);
  expect(t.dark, 'tokens redefined for the dark scheme').toEqual(t.base);
  expect(t.literals, 'colour values outside the token definitions').toEqual([]);
  expect(t.notColours, 'tokens whose value is not a valid colour').toEqual([]);
  await useScheme(page, 'dark');
  expect((await readTokens(page)).notColours, 'dark tokens whose value is not a valid colour').toEqual([]);
  const paired = new Set(PAIRS.flatMap(([fg, bg]) => [fg, bg]));
  expect(t.base.filter((x) => !paired.has(x)), 'tokens without a contrast pair').toEqual([]);
  expect([...paired].filter((x) => !t.base.includes(x)), 'pairs that name an unknown token').toEqual([]);
});

for (const scheme of SCHEMES) {
  test(`contrast matrix of the colour tokens, ${scheme} scheme`, async ({ page }, testInfo) => {
    await page.goto('/login');
    await useScheme(page, scheme);
    const { values } = await readTokens(page);
    const rows = PAIRS.map(([fg, bg, min, use]) => {
      const f = parseRgb(values[fg]);
      const b = parseRgb(values[bg]);
      expect(f.alpha, `--${fg} is opaque`).toBe(1);
      expect(b.alpha, `--${bg} is opaque`).toBe(1);
      return { fg, bg, min, use, fgValue: values[fg], bgValue: values[bg], ratio: Math.round(contrast(f.rgb, b.rgb) * 100) / 100 };
    });
    await testInfo.attach(`contrast-${scheme}.json`, { body: JSON.stringify(rows, null, 2), contentType: 'application/json' });
    // The light and the dark values must differ, or the dark scheme was not applied.
    if (scheme === 'dark') expect(values.bg, 'dark background').not.toBe('rgb(255, 255, 255)');
    const failing = rows.filter((r) => r.ratio < r.min).map((r) =>
      `--${r.fg} (${r.fgValue}) on --${r.bg} (${r.bgValue}): ${r.ratio}:1, needs ${r.min}:1 for ${r.use}`);
    expect(failing, `${scheme} scheme pairs below the minimum`).toEqual([]);
  });
}

// Guards against a false green: the stylesheet reader sees an extra token and colour literals when they exist, and
// the arithmetic matches the WCAG reference values.
test('token reader reports an untracked token and colour literals', async ({ page }) => {
  await page.goto('/login');
  await page.addStyleTag({ content: ':root{--extra:#000}.x{color:#123456}.y{border:1px solid red}.z{background:var(--bg)}' });
  const t = await readTokens(page);
  expect(t.base).toContain('extra');
  expect(t.dark).not.toContain('extra');
  expect(t.literals.some((l) => l.startsWith('.x '))).toBe(true);
  expect(t.literals.some((l) => l.startsWith('.y '))).toBe(true);
  expect(t.literals.some((l) => l.startsWith('.z '))).toBe(false);
});

test('contrast arithmetic matches known WCAG values', () => {
  expect(Math.round(contrast([0, 0, 0], [255, 255, 255]) * 100) / 100).toBe(21);
  expect(Math.round(contrast([118, 118, 118], [255, 255, 255]) * 100) / 100).toBe(4.54); // #767676 on white
  expect(Math.round(contrast([255, 255, 255], [255, 255, 255]) * 100) / 100).toBe(1);
});
