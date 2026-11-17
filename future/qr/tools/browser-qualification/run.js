// Usage: PLAYWRIGHT=/path/to/node_modules/playwright CHROMIUM=/path/to/chrome AXE_JS=/path/to/axe.min.js node run.js <dir-of-running-server>
// axe-core (MPL-2.0) and Playwright are external test tools: not vendored, not shipped.
const { chromium } = require(process.env.PLAYWRIGHT || "playwright");
const fs = require("fs");
const dir = process.argv[2];
const port = fs.readFileSync(dir + "/port", "utf8").trim();
const info = JSON.parse(fs.readFileSync(dir + "/info.json", "utf8"));
const axe = fs.readFileSync(process.env.AXE_JS, "utf8");
const base = "http://127.0.0.1:" + port;
(async () => {
  const browser = await chromium.launch({
    executablePath: process.env.CHROMIUM,
    args: ["--no-sandbox", "--disable-dev-shm-usage", "--use-fake-ui-for-media-stream", "--use-fake-device-for-media-stream", "--use-file-for-fake-video-capture=" + dir + "/cam.y4m"],
  });
  const results = {};
  async function ctx(member, opts = {}) {
    const c = await browser.newContext({ permissions: ["camera"], ...opts });
    await c.addCookies([{ name: "m", value: member, url: base }]);
    return c;
  }
  // 1. axe on each page (bypassCSP only to inject the test tool)
  const ax = await ctx("a", { bypassCSP: true });
  const pages = ["/qr", "/qr/scan", "/qr/print", "/qr/requests/" + info.id];
  results.axe = {};
  for (const p of pages) {
    const page = await ax.newPage();
    await page.goto(base + p);
    await page.addScriptTag({ content: axe });
    const r = await page.evaluate(async () => await axe.run(document, { runOnly: { type: "tag", values: ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa", "best-practice"] } }));
    results.axe[p] = { violations: r.violations.map((v) => ({ id: v.id, impact: v.impact, nodes: v.nodes.length, help: v.help })), passes: r.passes.length, incomplete: r.incomplete.map((v) => v.id) };
    await page.close();
  }
  // 2. camera flow as Bob: scan the fake camera frame showing Alice's request QR
  const bob = await ctx("b");
  const page = await bob.newPage();
  const errs = [];
  page.on("console", (m) => { if (m.type() === "error") errs.push(m.text()); });
  page.on("pageerror", (e) => errs.push("pageerror " + e.message));
  const csp = [];
  page.on("console", (m) => { if (/Content Security Policy|CSP/i.test(m.text())) csp.push(m.text()); });
  await page.goto(base + "/qr/scan");
  results.hasBarcodeDetector = await page.evaluate(() => "BarcodeDetector" in window);
  results.secureContext = await page.evaluate(() => window.isSecureContext);
  results.cameraSectionVisible = await page.isVisible("#camera-section");
  await page.click("#camera-start");
  try {
    await page.waitForURL("**/qr/v1/" + info.token, { timeout: 30000 });
    results.cameraScan = "navigated to confirmation";
    results.confirmText = (await page.textContent("main")).replace(/\s+/g, " ").slice(0, 400);
  } catch (e) {
    results.cameraScan = "FAILED: " + e.message.split("\n")[0];
    results.status = await page.textContent("#camera-status");
  }
  results.consoleErrors = errs;
  results.cspMessages = csp;
  // 3. no-JS: javascript disabled context, manual entry end to end
  const nojs = await ctx("b", { javaScriptEnabled: false });
  const np = await nojs.newPage();
  await np.goto(base + "/qr/scan");
  results.noJsCameraHidden = !(await np.isVisible("#camera-section"));
  await np.fill("#code", info.url);
  await Promise.all([np.waitForURL("**/qr/v1/" + info.token), np.click("#manual-form button[type=submit]")]);
  results.noJsConfirm = (await np.textContent("h1")).trim();
  // keyboard-only: tab order reaches the submit button and Enter submits
  await np.goto(base + "/qr/scan");
  await np.keyboard.press("Tab"); // skip link
  await np.keyboard.press("Tab");
  results.keyboardFocus = [];
  for (let i = 0; i < 6; i++) { results.keyboardFocus.push(await np.evaluate(() => document.activeElement.tagName + ":" + (document.activeElement.id || document.activeElement.textContent.trim().slice(0, 20)))); await np.keyboard.press("Tab"); }
  // 4. print stylesheet hides chrome
  const pr = await ax.newPage();
  await pr.goto(base + "/qr/print");
  await pr.emulateMedia({ media: "print" });
  results.printHidesNav = !(await pr.isVisible("nav"));
  results.printQrVisible = await pr.isVisible("img.qr-print");
  await pr.screenshot({ path: dir + "/print.png" });
  // 5. mobile viewport overflow check
  const mob = await browser.newContext({ viewport: { width: 360, height: 640 } });
  await mob.addCookies([{ name: "m", value: "a", url: base }]);
  const mp = await mob.newPage();
  results.mobileOverflow = {};
  for (const p of ["/qr", "/qr/scan"]) { await mp.goto(base + p); results.mobileOverflow[p] = await mp.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth); }
  console.log(JSON.stringify(results, null, 1));
  await browser.close();
})().catch((e) => { console.error("HARNESS ERROR", e.message); process.exit(1); });
