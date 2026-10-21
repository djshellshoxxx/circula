// A second, self-contained instance for the security pages. It has its own database so that changing a password,
// turning on two-step sign-in or erasing a member never disturbs the shared seed used by the other specs.
// Seeding goes through the web forms; the operator-only settings (dual control, break-glass) go through the CLI
// while the server is stopped, because the CLI takes the single-writer lock.
const fs = require('fs');
const os = require('os');
const path = require('path');
const crypto = require('crypto');
const { execFileSync } = require('child_process');
const { chromium } = require('@playwright/test');
const { chromiumExecutable } = require('./launch');
const gs = require('./global-setup');

const { register, login, submit, PASSWORD, SETUP_SECRET, SLUG } = gs.helpers;
const MFA_SECRET = 'e2e-mfa-secret-0123456789abcdefghij';

const bin = () => path.resolve(process.env.CIRCULA_BIN || path.join(__dirname, '..', 'bin', 'circula'));

function b32decode(s) {
  const alpha = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ234567';
  let bits = '';
  for (const ch of s.replace(/[\s=]/g, '').toUpperCase()) bits += alpha.indexOf(ch).toString(2).padStart(5, '0');
  const out = [];
  for (let i = 0; i + 8 <= bits.length; i += 8) out.push(parseInt(bits.slice(i, i + 8), 2));
  return Buffer.from(out);
}

/** RFC 6238 code (SHA-1, 6 digits, 30 s) for the given base32 secret; stepOffset 1 gives the next window's code. */
function totp(secret, stepOffset = 0) {
  const step = Math.floor(Date.now() / 30000) + stepOffset;
  const buf = Buffer.alloc(8);
  buf.writeBigUInt64BE(BigInt(step));
  const h = crypto.createHmac('sha1', b32decode(secret)).update(buf).digest();
  const o = h[h.length - 1] & 0xf;
  const n = ((h[o] & 0x7f) << 24) | (h[o + 1] << 16) | (h[o + 2] << 8) | h[o + 3];
  return String(n % 1000000).padStart(6, '0');
}

function cli(dir, args) {
  return execFileSync(bin(), args, {
    env: {
      PATH: process.env.PATH,
      CIRCULA_DATABASE: path.join(dir, 'circula.db'),
      CIRCULA_BACKUP_DIR: path.join(dir, 'backups'),
      CIRCULA_MIGRATE_MODE: 'auto',
      CIRCULA_MFA_SECRET: MFA_SECRET,
    },
    encoding: 'utf8',
  });
}

const start = (dir) => gs.startInstance({ CIRCULA_MFA_SECRET: MFA_SECRET }, { dir });

/**
 * Builds the instance. Returns { dir, inst, users, restart(), stop(), cleanup() } where users maps
 * admin, admin2, alice, bob, dora, eve to login names. Everyone has PASSWORD.
 */
async function buildSecurityInstance() {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'circula-e2e-sec-'));
  let inst = await start(dir);
  const users = { admin: 'sadmin@example.org', admin2: 'sadmin2@example.org', alice: 'salice@example.org', bob: 'sbob@example.org', dora: 'sdora@example.org', eve: 'seve@example.org' };
  const browser = await chromium.launch({ executablePath: chromiumExecutable(), args: ['--no-sandbox', '--disable-gpu'] });
  try {
    const ctx = () => browser.newContext({ baseURL: inst.base });
    let c = await ctx();
    let p = await c.newPage();
    await register(p, users.admin);
    await login(p, users.admin);
    await p.goto('/setup');
    await p.getByLabel('Setup secret').fill(SETUP_SECRET);
    await p.getByLabel('Network name').fill('Secure Circle');
    await p.getByLabel('Web address name').fill(SLUG);
    await p.getByLabel('Your display name in this network').fill('Sam Admin');
    await p.getByLabel('Credit unit name').fill('credit');
    await p.getByLabel('Default credit limit per member').fill('100.00');
    await p.getByLabel('Sign-up grant').fill('0.00');
    await submit(p, 'Create network');
    const links = [];
    for (let i = 0; i < 5; i++) {
      await p.goto(`/n/${SLUG}/admin/invitations`);
      await p.getByLabel('Valid for (days)').fill('7');
      await submit(p, 'Create invitation link');
      links.push(new URL((await p.locator('#flash code').innerText()).trim()).pathname);
    }
    const people = [['admin2', 'Ada Second'], ['alice', 'Alice Member'], ['bob', 'Bob Member'], ['dora', 'Dora Leaver'], ['eve', 'Eve Leaver']];
    for (let i = 0; i < people.length; i++) {
      const [key, display] = people[i];
      const cc = await ctx();
      const pp = await cc.newPage();
      await register(pp, users[key]);
      await login(pp, users[key]);
      await pp.goto(links[i]);
      await pp.getByLabel('Your display name').fill(display);
      await submit(pp, 'Join the network');
      await cc.close();
    }
    // The founder makes the second administrator (dual control needs two).
    await p.goto(`/n/${SLUG}/directory`);
    const adaHref = await p.locator(`a[href*="/n/${SLUG}/members/"]`).filter({ hasText: 'Ada' }).first().getAttribute('href');
    await p.goto(`/n/${SLUG}/admin/members/${adaHref.split('/').pop()}/grant-admin`);
    await submit(p, 'Make administrator');
    await c.close();
  } finally {
    await browser.close();
  }
  await inst.stop();
  // Operator settings, with the server stopped.
  cli(dir, ['dual-control', '-network', SLUG, '-threshold-minor', '100', '-reason', 'e2e dual control']);
  cli(dir, ['break-glass', 'open', '-network', SLUG, '-reason', 'e2e support session', '-operator', 'ops-e2e']);
  inst = await start(dir);

  const self = {
    dir, users, get inst() { return inst; }, get base() { return inst.base; },
    async restart() { await inst.stop(); inst = await start(dir); return inst; },
    async stop() { await inst.stop(); },
    async cleanup() { try { await inst.stop(); } finally { fs.rmSync(dir, { recursive: true, force: true }); } },
    /** Stops the server, changes a cached balance so ledger verification fails, and starts again: the network is frozen. */
    async restartWithBrokenLedger() {
      await inst.stop();
      const { DatabaseSync } = require('node:sqlite');
      const db = new DatabaseSync(path.join(dir, 'circula.db'));
      db.exec('UPDATE accounts SET balance = balance + 7 WHERE rowid = (SELECT MIN(rowid) FROM accounts)');
      db.close();
      inst = await start(dir);
      return inst;
    },
  };
  return self;
}

module.exports = { buildSecurityInstance, totp, PASSWORD, SLUG };
