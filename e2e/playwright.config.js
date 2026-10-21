// Playwright configuration for the accessibility and keyboard end-to-end tests.
// Browser: the preinstalled headless Chromium (no download). Override the path with CHROMIUM_PATH.
const path = require('path');
const { defineConfig } = require('@playwright/test');
const { chromiumExecutable } = require('./launch');

module.exports = defineConfig({
  testDir: path.join(__dirname, 'tests'),
  globalSetup: require.resolve('./global-setup.js'),
  // The tests share one seeded database and the keyboard tests change it, so run serially.
  workers: 1,
  fullyParallel: false,
  retries: 0,
  timeout: 60_000,
  reporter: [['list']],
  outputDir: path.join(__dirname, 'test-results'),
  use: {
    headless: true,
    bypassCSP: true, // axe-core is injected as a script; the app's CSP forbids inline scripts
    launchOptions: {
      executablePath: chromiumExecutable(),
      args: ['--no-sandbox', '--disable-gpu', '--disable-dev-shm-usage'],
    },
    trace: 'off',
    screenshot: 'only-on-failure',
  },
});
