// Locates the Chromium executable shared by the config, the global setup and the tests.
const fs = require('fs');

function chromiumExecutable() {
  const candidates = [
    process.env.CHROMIUM_PATH,
    '/opt/pw-browsers/chromium-1194/chrome-linux/chrome',
    '/opt/pw-browsers/chromium',
    '/usr/bin/chromium',
    '/usr/bin/chromium-browser',
    '/usr/bin/google-chrome',
  ].filter(Boolean);
  for (const c of candidates) {
    try {
      if (fs.statSync(c).isFile()) return c;
    } catch (_) { /* try the next one */ }
  }
  // CI: `npx playwright install chromium` puts the pinned browser in Playwright's own cache.
  try {
    const own = require('@playwright/test').chromium.executablePath();
    if (own && fs.statSync(own).isFile()) return own;
  } catch (_) { /* none installed */ }
  throw new Error('no Chromium found; set CHROMIUM_PATH (tried: ' + candidates.join(', ') + ')');
}

module.exports = { chromiumExecutable };
