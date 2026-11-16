// Target size measurement for UX-035 (WCAG 2.2 success criterion 2.5.8, Target Size (Minimum)), shared by
// tests/target-size.spec.js (every page of the shared seed) and tests/a11y-governance.spec.js (its own instance).
//
// MUST: every pointer target is at least 24 by 24 CSS pixels, or is exempt the way 2.5.8 allows: an inline link inside
// a line of text, or an undersized target whose 24 px circle (centred on it) reaches no other target and no other
// undersized target's circle (the spacing exception).
// SHOULD (touch layouts, checked when `comfortable` is set): the controls the stylesheet sizes for touch are at least
// 44 by 44 CSS pixels: buttons, button-styled links, text fields, menus, the main navigation links, and each checkbox
// or radio button together with its label (the label is part of the target).

/** Runs in the page. Returns { count, failures (24 px), comfy (44 px) } for the current layout. */
function measure(comfortable) {
  const TARGETS = 'a[href], button, input:not([type=hidden]), select, textarea, summary, [role=button], [tabindex]:not([tabindex="-1"])';
  const describe = (el) => `${el.tagName.toLowerCase()}${el.id ? '#' + el.id : ''}${el.className ? '.' + String(el.className).split(' ').join('.') : ''} "${(el.innerText || el.value || el.name || '').replace(/\s+/g, ' ').trim().slice(0, 40)}"`;
  const onScreen = (el) => {
    const r = el.getBoundingClientRect();
    const cs = getComputedStyle(el);
    return r.width > 0 && r.height > 0 && cs.visibility !== 'hidden' && r.right > 0 && r.bottom > 0;
  };
  // A checkbox or radio button wrapped in its label: the label is part of the target.
  const targetEl = (el) => (el.matches('input[type=checkbox], input[type=radio]') && el.closest('label')) || el;
  const els = [...document.querySelectorAll(TARGETS)].filter(onScreen);
  const boxes = new Map(els.map((el) => [el, targetEl(el).getBoundingClientRect()]));
  // A link that wraps onto a second line is two boxes, one per line; its bounding box would span both lines.
  const parts = new Map(els.map((el) => [el, [...targetEl(el).getClientRects()].filter((r) => r.width > 0 && r.height > 0)]));
  const blockOf = (el) => {
    let b = el.parentElement;
    while (b && getComputedStyle(b).display.startsWith('inline')) b = b.parentElement;
    return b;
  };
  // 2.5.8 inline exception: the link sits in a line of text (its block holds text that belongs to no target).
  const inline = (el) => {
    if (getComputedStyle(el).display !== 'inline') return false;
    const block = blockOf(el);
    if (!block) return false;
    let text = block.innerText;
    for (const t of block.querySelectorAll(TARGETS)) text = text.split(t.innerText).join(' ');
    return /[\p{L}\p{N}·]/u.test(text);
  };
  const tooSmall = (r) => r.width < 24 || r.height < 24;
  // undersized boxes, each with the target it belongs to
  const small = els.filter((el) => !inline(el)).flatMap((el) => parts.get(el).filter(tooSmall).map((r) => ({ el, r })));
  const centre = (r) => [r.left + r.width / 2, r.top + r.height / 2];
  const distToRect = ([x, y], r) => Math.hypot(Math.max(r.left - x, 0, x - r.right), Math.max(r.top - y, 0, y - r.bottom));
  const failures = [];
  for (const { el, r } of small) {
    const c = centre(r);
    const unrelated = (o) => o !== el && !o.contains(el) && !el.contains(o);
    // the 24 px circle reaches another target, or intersects the circle of another undersized target
    const reach = els.find((o) => unrelated(o) && parts.get(o).some((ro) => distToRect(c, ro) < 12));
    const cross = small.find((s) => unrelated(s.el) && Math.hypot(c[0] - centre(s.r)[0], c[1] - centre(s.r)[1]) < 24);
    const clash = reach || (cross && cross.el);
    if (clash) failures.push(`${describe(el)} is ${Math.round(r.width)}x${Math.round(r.height)} and too close to ${describe(clash)}`);
  }
  const comfy = [];
  if (comfortable) {
    const sized = 'button, .btn, input:not([type=hidden]), select, textarea, ul.nav a';
    for (const el of els.filter((e) => e.matches(sized))) {
      const r = boxes.get(el);
      if (r.width < 44 || r.height < 44) comfy.push(`${describe(el)} is ${Math.round(r.width)}x${Math.round(r.height)}`);
    }
  }
  return { count: els.length, failures, comfy };
}

/** The UX-035 failures of the current page at its current layout, as readable lines. */
async function targetSizeFailures(page, comfortable) {
  const r = await page.evaluate(measure, !!comfortable);
  return {
    count: r.count,
    failures: [...r.failures.map((f) => `below 24 px: ${f}`), ...r.comfy.map((f) => `below 44 px on a touch layout: ${f}`)],
  };
}

/**
 * Measures the current page at its own viewport (24 px) and at a 320 px wide layout (24 and 44 px), then restores the
 * viewport. The layout does not depend on touch input, so width alone gives the touch layout here.
 */
async function expectTargetSizes(page, label) {
  const size = page.viewportSize();
  const failures = (await targetSizeFailures(page, false)).failures.map((f) => `${size.width} px: ${f}`);
  await page.setViewportSize({ width: 320, height: 640 });
  failures.push(...(await targetSizeFailures(page, true)).failures.map((f) => `320 px: ${f}`));
  await page.setViewportSize(size);
  if (failures.length) throw new Error(`${label}: target sizes (UX-035) at ${page.url()}\n  ${failures.join('\n  ')}`);
}

module.exports = { measure, targetSizeFailures, expectTargetSizes };
