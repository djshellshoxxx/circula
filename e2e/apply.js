// The shared fixtures plus pages in the second seeded instance ("Apply Circle", see seedApply in global-setup.js):
// people join it by request and administrator approval, and it has published terms.
const path = require('path');
const fixtures = require('./fixtures');

const PREFIX = 'apply:';

/** Context options for a row's role: "apply:<name>" is the approval-mode instance, other names the main seed, null signed out. */
function session(st, who) {
  if (who && who.startsWith(PREFIX)) {
    return { baseURL: st.apply.base, storageState: path.join(fixtures.STATE_DIR, `apply-${who.slice(PREFIX.length)}.json`) };
  }
  return who ? { baseURL: st.base, storageState: path.join(fixtures.STATE_DIR, who + '.json') } : { baseURL: st.base };
}

/** The Home path of the network a row's role belongs to. */
const homeOf = (st, who) => `/n/${who && who.startsWith(PREFIX) ? st.apply.slug : st.slug}`;

const test = fixtures.test.extend({
  /** Opens a new page in the approval-mode instance signed in as erin, hana, finn or gail. */
  asApply: async ({ browser, st }, use) => {
    const contexts = [];
    await use(async (who) => {
      const ctx = await browser.newContext({ ...session(st, PREFIX + who), bypassCSP: true });
      contexts.push(ctx);
      return ctx.newPage();
    });
    for (const c of contexts) await c.close();
  },
  /** Opens a signed-in page for a row's role, in whichever instance it belongs to. */
  open: async ({ as, asApply }, use) => {
    await use((who) => (who.startsWith(PREFIX) ? asApply(who.slice(PREFIX.length)) : as(who)));
  },
});

module.exports = { ...fixtures, test, session, homeOf };
