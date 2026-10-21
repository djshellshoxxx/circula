// Boots the real circula binary on a free loopback port with a temporary database, seeds it through
// the web forms (the same way a person would), and records what the tests need in .state/state.json.
// The returned function stops the binary and removes the temporary directory.
const fs = require('fs');
const os = require('os');
const net = require('net');
const path = require('path');
const { spawn } = require('child_process');
const { chromium } = require('@playwright/test');

const STATE_DIR = path.join(__dirname, '.state');
const STATE_FILE = path.join(STATE_DIR, 'state.json');

const PASSWORD = 'correct horse battery staple';
const SLUG = 'demo';
const SETUP_SECRET = 'e2e-setup-secret-0123456789abcdef';

function freePort() {
  return new Promise((resolve, reject) => {
    const s = net.createServer();
    s.listen(0, '127.0.0.1', () => {
      const { port } = s.address();
      s.close(() => resolve(port));
    });
    s.on('error', reject);
  });
}

async function waitReady(base, child, logs) {
  const deadline = Date.now() + 30_000;
  while (Date.now() < deadline) {
    if (child.exitCode !== null) throw new Error('circula exited early:\n' + logs.join(''));
    try {
      const r = await fetch(base + '/healthz');
      if (r.ok) return;
    } catch (_) { /* not up yet */ }
    await new Promise((r) => setTimeout(r, 100));
  }
  throw new Error('circula did not become ready:\n' + logs.join(''));
}

async function submit(page, name) {
  await Promise.all([page.waitForLoadState('load'), page.getByRole('button', { name, exact: true }).click()]);
}

async function register(page, login) {
  await page.goto('/register');
  await page.getByLabel('Login', { exact: true }).fill(login);
  await page.getByLabel('Password', { exact: true }).fill(PASSWORD);
  await page.getByLabel('Confirm password').fill(PASSWORD);
  await submit(page, 'Create account');
}

async function login(page, loginName, next) {
  await page.goto('/login' + (next ? '?next=' + encodeURIComponent(next) : ''));
  await page.getByLabel('Login', { exact: true }).fill(loginName);
  await page.getByLabel('Password', { exact: true }).fill(PASSWORD);
  await submit(page, 'Sign in');
}

/** Starts one more circula instance with a fresh temporary database. env adds or overrides settings. */
async function startInstance(env = {}, opts = {}) {
  const bin = path.resolve(process.env.CIRCULA_BIN || path.join(__dirname, '..', 'bin', 'circula'));
  if (!fs.existsSync(bin)) throw new Error(`binary not found: ${bin} (run "make build" or set CIRCULA_BIN)`);
  // opts.dir reuses a directory (and its database) the caller owns; stop() then leaves it in place.
  const dir = opts.dir || fs.mkdtempSync(path.join(os.tmpdir(), 'circula-e2e-'));
  const port = await freePort();
  const base = `http://127.0.0.1:${port}`;
  const logs = [];
  const child = spawn(bin, ['serve'], {
    env: {
      PATH: process.env.PATH,
      CIRCULA_LISTEN: `127.0.0.1:${port}`,
      CIRCULA_PUBLIC_BASE_URL: base,
      CIRCULA_DATABASE: path.join(dir, 'circula.db'),
      CIRCULA_BACKUP_DIR: path.join(dir, 'backups'),
      CIRCULA_BACKUP_INTERVAL: '0',
      CIRCULA_MIGRATE_MODE: 'auto',
      CIRCULA_COOKIE_SECURE: 'false', // plain http on loopback
      CIRCULA_LOG_FORMAT: 'text',
      CIRCULA_SETUP_SECRET: SETUP_SECRET,
      ...env,
    },
    stdio: ['ignore', 'pipe', 'pipe'],
  });
  child.stdout.on('data', (d) => logs.push(d.toString()));
  child.stderr.on('data', (d) => logs.push(d.toString()));
  const stop = async () => {
    if (child.exitCode === null) {
      child.kill('SIGTERM');
      await new Promise((r) => {
        const t = setTimeout(() => { child.kill('SIGKILL'); r(); }, 10_000);
        child.on('exit', () => { clearTimeout(t); r(); });
      });
    }
    if (!opts.dir) fs.rmSync(dir, { recursive: true, force: true });
  };
  try {
    await waitReady(base, child, logs);
  } catch (e) {
    await stop();
    throw e;
  }
  return { base, dir, stop, logs };
}

module.exports = async function globalSetup() {
  const bin = path.resolve(process.env.CIRCULA_BIN || path.join(__dirname, '..', 'bin', 'circula'));
  if (!fs.existsSync(bin)) throw new Error(`binary not found: ${bin} (run "make build" or set CIRCULA_BIN)`);

  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'circula-e2e-'));
  const port = await freePort();
  const base = `http://127.0.0.1:${port}`;
  const logs = [];
  const child = spawn(bin, ['serve'], {
    env: {
      PATH: process.env.PATH,
      CIRCULA_LISTEN: `127.0.0.1:${port}`,
      CIRCULA_PUBLIC_BASE_URL: base,
      CIRCULA_DATABASE: path.join(dir, 'circula.db'),
      CIRCULA_BACKUP_DIR: path.join(dir, 'backups'),
      CIRCULA_BACKUP_INTERVAL: '0',
      CIRCULA_MIGRATE_MODE: 'auto',
      CIRCULA_COOKIE_SECURE: 'false', // plain http on loopback
      CIRCULA_LOG_FORMAT: 'text',
      CIRCULA_SETUP_SECRET: SETUP_SECRET,
    },
    stdio: ['ignore', 'pipe', 'pipe'],
  });
  child.stdout.on('data', (d) => logs.push(d.toString()));
  child.stderr.on('data', (d) => logs.push(d.toString()));

  const teardown = async () => {
    if (child.exitCode === null) {
      child.kill('SIGTERM');
      await new Promise((r) => {
        const t = setTimeout(() => { child.kill('SIGKILL'); r(); }, 10_000);
        child.on('exit', () => { clearTimeout(t); r(); });
      });
    }
    fs.rmSync(dir, { recursive: true, force: true });
    fs.rmSync(STATE_DIR, { recursive: true, force: true });
  };

  try {
    await waitReady(base, child, logs);
    fs.mkdirSync(STATE_DIR, { recursive: true });
    const state = await seed(base);
    state.base = base;
    state.slug = SLUG;
    state.password = PASSWORD;
    fs.writeFileSync(STATE_FILE, JSON.stringify(state, null, 2));
  } catch (e) {
    await teardown();
    throw e;
  }
  return teardown;
};

async function seed(base) {
  const { chromiumExecutable } = require('./launch');
  const browser = await chromium.launch({ executablePath: chromiumExecutable(), args: ['--no-sandbox', '--disable-gpu'] });
  const ctx = (opts = {}) => browser.newContext({ baseURL: base, ...opts });
  const state = { users: { admin: 'admin@example.org', alice: 'alice@example.org', bob: 'bob@example.org', carol: 'carol@example.org' } };
  try {
    // 1. Founder: register, sign in, create the network.
    let c = await ctx();
    let p = await c.newPage();
    await register(p, state.users.admin);
    await login(p, state.users.admin);
    await p.goto('/setup');
    await p.getByLabel('Setup secret').fill(SETUP_SECRET);
    await p.getByLabel('Network name').fill('Demo Circle');
    await p.getByLabel('Web address name').fill(SLUG);
    await p.getByLabel('Your display name in this network').fill('Dana Admin');
    await p.getByLabel('Credit unit name').fill('credit');
    await p.getByLabel('Default credit limit per member').fill('100.00');
    await p.getByLabel('Sign-up grant').fill('0.00');
    await submit(p, 'Create network');
    if (!new URL(p.url()).pathname.startsWith(`/n/${SLUG}`)) throw new Error('network not created, at ' + p.url());
    await c.storageState({ path: path.join(STATE_DIR, 'admin.json') });

    // 2. Invitations: one per member plus one left unused for the anonymous join page.
    const links = [];
    for (let i = 0; i < 4; i++) {
      await p.goto(`/n/${SLUG}/admin/invitations`);
      await p.getByLabel('Valid for (days)').fill('7');
      await submit(p, 'Create invitation link');
      const link = (await p.locator('#flash code').innerText()).trim();
      links.push(link);
    }
    state.unusedInvite = new URL(links[3]).pathname;
    await c.close();

    // 3. Members join through their links.
    const people = [['alice', 'Alice Member'], ['bob', 'Bob Member'], ['carol', 'Carol Member']];
    for (let i = 0; i < people.length; i++) {
      const [key, display] = people[i];
      c = await ctx();
      p = await c.newPage();
      await register(p, state.users[key]);
      await login(p, state.users[key]);
      await p.goto(new URL(links[i]).pathname);
      await p.getByLabel('Your display name').fill(display);
      await submit(p, 'Join the network');
      await c.storageState({ path: path.join(STATE_DIR, key + '.json') });
      await c.close();
    }

    // 4. Trades: alice pays bob 10.00 (completed), carol proposes 5.00 to bob (left pending),
    //    alice pays carol 2.50 (completed, then disputed).
    const as = async (who) => {
      const cc = await ctx({ storageState: path.join(STATE_DIR, who + '.json') });
      return [cc, await cc.newPage()];
    };
    const propose = async (pg, to, amount, note, direction) => {
      await pg.goto(`/n/${SLUG}/trades/new`);
      await pg.getByLabel('Who is the trade with?').selectOption({ label: to });
      if (direction === 'request') await pg.getByLabel('They pay me').check();
      await pg.getByLabel('Amount').fill(amount);
      await pg.getByLabel('What is it for? (optional)').fill(note);
      await submit(pg, 'Review trade');
      await submit(pg, 'Send trade');
      const m = new URL(pg.url()).pathname.match(/\/trades\/([^/]+)$/);
      if (!m) throw new Error('propose did not land on a trade page: ' + pg.url());
      return m[1];
    };
    const accept = async (pg, id) => {
      await pg.goto(`/n/${SLUG}/trades/${id}/accept`);
      await submit(pg, 'Confirm and accept');
    };

    let [ca, pa] = await as('alice');
    let [cb, pb] = await as('bob');
    let [cc, pc] = await as('carol');
    const t1 = await propose(pa, 'Bob Member', '10.00', 'garden help', 'pay');
    await accept(pb, t1);
    const t2 = await propose(pa, 'Carol Member', '2.50', 'bike repair', 'pay');
    await accept(pc, t2);
    const t3 = await propose(pc, 'Bob Member', '5.00', 'lawn mowing', 'pay'); // stays proposed
    state.tradeDone = t1;
    state.tradeDisputed = t2;
    state.tradePending = t3; // carol -> bob, bob can accept

    // 5. A listing by alice.
    await pa.goto(`/n/${SLUG}/listings/new`);
    await pa.getByLabel('Title').fill('Bike repair');
    await pa.getByLabel('Description').fill('Puncture fixes and basic tune-ups.');
    await pa.getByLabel('Price (optional)').fill('5.00');
    await submit(pa, 'Publish listing');
    await pa.goto(`/n/${SLUG}/listings`);
    const edit = await pa.locator('a[href*="/listings/"][href$="/edit"]').first().getAttribute('href');
    state.listingEdit = edit;

    // 6. A dispute by alice on the bike repair trade.
    await pa.goto(`/n/${SLUG}/trades/${t2}/dispute`);
    await pa.getByLabel('What went wrong?').fill('The repair did not hold.');
    await submit(pa, 'Review dispute');
    await submit(pa, 'Open dispute');
    await pa.goto(`/n/${SLUG}/disputes`);
    const d = await pa.locator(`a[href*="/n/${SLUG}/disputes/"]`).first().getAttribute('href');
    state.disputePath = d;

    // 7. Member id of bob, for profile and admin action pages.
    await pa.goto(`/n/${SLUG}/directory`);
    const prof = await pa.locator(`a[href*="/n/${SLUG}/members/"]`).filter({ hasText: 'Bob' }).first().getAttribute('href');
    state.bobProfile = prof;
    state.bobId = prof.split('/').pop();

    // 8. More trades: t4 and t5 are completed alice -> bob payments. t4 is disputed below, t5 is left for the reversal pages.
    const t4 = await propose(pa, 'Bob Member', '1.00', 'bike tuning', 'pay');
    await accept(pb, t4);
    const t5 = await propose(pa, 'Bob Member', '0.50', 'spare key', 'pay');
    await accept(pb, t5);
    state.tradeReverse = t5;

    // 9. Admin: make carol a dispute reviewer, publish an announcement.
    let [cd, pd] = await as('admin');
    await pa.goto(`/n/${SLUG}/directory`);
    const carolHref = await pa.locator(`a[href*="/n/${SLUG}/members/"]`).filter({ hasText: 'Carol' }).first().getAttribute('href');
    state.carolId = carolHref.split('/').pop();
    await pd.goto(`/n/${SLUG}/admin/members/${state.carolId}/grant-reviewer`);
    await submit(pd, 'Make dispute reviewer');
    await pd.goto(`/n/${SLUG}/admin/announcements`);
    await pd.getByLabel('Title').fill('Meet-up on Saturday');
    await pd.getByLabel('Message').fill('We meet at the library at noon to swap goods and talk.');
    await submit(pd, 'Publish announcement');

    // 10. A report by carol about alice's listing.
    const listingId = edit.split('/').slice(-2)[0];
    state.listingId = listingId;
    state.reportPath = `/n/${SLUG}/report?type=listing&id=${listingId}`;
    await pc.goto(state.reportPath);
    await submit(pc, 'Send report');

    // 11. Disputes with a response and re-review. The ids of new disputes are read from the list.
    const disputeIds = async () => {
      await pa.goto(`/n/${SLUG}/disputes`);
      return (await pa.locator(`a[href*="/n/${SLUG}/disputes/"]`).evaluateAll((a) => a.map((x) => x.getAttribute('href'))));
    };
    const openDispute = async (trade, text) => {
      const before = await disputeIds();
      await pa.goto(`/n/${SLUG}/trades/${trade}/dispute`);
      await pa.getByLabel('What went wrong?').fill(text);
      await submit(pa, 'Review dispute');
      await submit(pa, 'Open dispute');
      const after = await disputeIds();
      const fresh = after.find((h) => !before.includes(h));
      if (!fresh) throw new Error('dispute not created for ' + trade);
      return fresh;
    };
    const decide = async (pg, path, outcomeLabel, why) => {
      await pg.goto(path);
      await submit(pg, 'Start review');
      await pg.getByLabel('Outcome').selectOption({ label: outcomeLabel });
      await pg.getByLabel('Rationale').fill(why);
      await submit(pg, 'Review decision');
      await submit(pg, 'Record decision');
    };
    // D2 (t4): bob responds, admin dismisses; alice can still ask for a re-review (state: resolved, with response).
    state.disputeResolved = await openDispute(t4, 'The tuning was not finished.');
    await pb.goto(state.disputeResolved);
    await pb.getByLabel('Your response').fill('It was finished and tested before hand-over.');
    await submit(pb, 'Send response');
    await decide(pd, state.disputeResolved, 'Dismissed: the balance stands', 'The work was delivered as described.');
    // D3 (t1): admin dismisses, alice asks for a re-review; carol (reviewer, not a party) can decide it.
    state.disputeReReview = await openDispute(t1, 'I did not agree to the final amount.');
    await decide(pd, state.disputeReReview, 'Dismissed: the balance stands', 'The amount matches what both agreed.');
    await pa.goto(state.disputeReReview);
    await pa.getByLabel('Why should it be looked at again?').fill('The chat shows a lower amount was agreed.');
    await submit(pa, 'Request re-review');

    await ca.close();
    await cb.close();
    await cc.close();
    await cd.close();
  } finally {
    await browser.close();
  }
  return state;
}

module.exports.startInstance = startInstance;
module.exports.helpers = { register, login, submit, PASSWORD, SETUP_SECRET, SLUG };
module.exports.STATE_FILE = STATE_FILE;
module.exports.STATE_DIR = STATE_DIR;
