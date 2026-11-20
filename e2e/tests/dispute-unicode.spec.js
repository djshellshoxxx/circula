// FR-070/073/078: native text entry must allow the same code-point boundary as
// the service. Create a new trade so the shared dispute fixtures stay available.
const { test, expect } = require('../fixtures');

async function enterUnicodeText(page, label, name, text, hint) {
  const field = page.getByLabel(label, { exact: true });
  expect(await field.getAttribute('maxlength')).toBeNull();
  await expect(field).toHaveAttribute('required', '');
  await expect(field).toHaveAttribute('aria-describedby', `h-${name}`);
  await expect(page.locator(`#h-${name}`)).toHaveText(hint);
  await field.fill(text);
  await expect(field).toHaveValue(text);
}

test('dispute statements and both rationales preserve 2000 emoji through confirmation', async ({ as, st }) => {
  const alice = await as('alice');
  const bob = await as('bob');
  const admin = await as('admin');
  const carol = await as('carol');
  const base = `/n/${st.slug}`;
  const submit = (page, name) => page.getByRole('button', { name, exact: true }).click();

  await alice.goto(`${base}/trades/new`);
  await alice.getByLabel('Who is the trade with?', { exact: true }).selectOption(st.bobId);
  await alice.getByLabel('Amount', { exact: true }).fill('0.01');
  await submit(alice, 'Review trade');
  await submit(alice, 'Send trade');
  await expect(alice).toHaveURL(new RegExp(`${base}/trades/[^/]+$`));
  const tradePath = new URL(alice.url()).pathname;
  await bob.goto(`${tradePath}/accept`);
  await submit(bob, 'Confirm and accept');
  await expect(bob).toHaveURL((url) => url.pathname === tradePath);

  const statement = '😀'.repeat(2000);
  await alice.goto(`${tradePath}/dispute`);
  await enterUnicodeText(alice, 'What went wrong?', 'statement', statement,
    'Up to 2000 characters. The other party and the reviewer will read this.');
  await submit(alice, 'Review dispute');
  await expect(alice.locator('input[name="statement"]')).toHaveValue(statement);
  await submit(alice, 'Open dispute');
  await expect(alice).toHaveURL(new RegExp(`${base}/disputes/[^/]+$`));
  const disputePath = new URL(alice.url()).pathname;
  await expect(alice.locator('main')).toContainText(statement);

  await bob.goto(disputePath);
  await bob.getByLabel('Your response', { exact: true }).fill('Please review the recorded agreement.');
  await submit(bob, 'Send response');
  await expect(bob.locator('#flash')).toContainText('Your response was added');

  const rationale = '🧭'.repeat(2000);
  await admin.goto(disputePath);
  await submit(admin, 'Start review');
  await admin.getByLabel('Outcome', { exact: true }).selectOption('dismissed');
  await enterUnicodeText(admin, 'Rationale', 'rationale', rationale, 'Required. Both parties will see it.');
  await submit(admin, 'Review decision');
  await expect(admin.locator('input[name="rationale"]')).toHaveValue(rationale);
  await submit(admin, 'Record decision');
  await expect(admin).toHaveURL((url) => url.pathname === disputePath);
  await expect(admin.locator('main')).toContainText(rationale);

  await alice.goto(disputePath);
  await alice.getByLabel('Why should it be looked at again?', { exact: true }).fill('Please review the additional context.');
  await submit(alice, 'Request re-review');
  await expect(alice.locator('#flash')).toContainText('Re-review requested');

  const reReviewRationale = '🌿'.repeat(2000);
  await carol.goto(disputePath);
  await carol.getByLabel('Outcome', { exact: true }).selectOption('confirmed');
  await enterUnicodeText(carol, 'Rationale', 'rationale', reReviewRationale, 'Required. Both parties will see it.');
  await submit(carol, 'Review re-review decision');
  await expect(carol.locator('input[name="rationale"]')).toHaveValue(reReviewRationale);
  await submit(carol, 'Record re-review decision');
  await expect(carol).toHaveURL((url) => url.pathname === disputePath);
  await expect(carol.locator('main')).toContainText(reReviewRationale);
});
