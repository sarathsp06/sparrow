import { expect, request as playwrightRequest, test, type APIRequestContext, type Page } from '@playwright/test';
import { mintPortalToken, newConsumer, registerWebhook } from '../api';

// Split deployment: the built UI is served by a plain static host (no Sparrow
// involvement) on one origin, and talks cross-origin to a Sparrow server that
// runs with ENVIRONMENT=production, SPARROW_API_KEY and CORS_ALLOWED_ORIGINS.
// scripts/test-ui.sh boots both and sets these variables.
const UI = process.env.SPARROW_SPLIT_UI_URL ?? '';
const API = process.env.SPARROW_SPLIT_API_URL ?? '';
const KEY = process.env.SPARROW_SPLIT_API_KEY ?? '';

test.skip(!UI || !API || !KEY, 'split-deployment stack not running (see scripts/test-ui.sh)');

let admin: APIRequestContext;
test.beforeAll(async () => {
  admin = await playwrightRequest.newContext({ baseURL: API, extraHTTPHeaders: { 'X-API-Key': KEY } });
});
test.afterAll(async () => admin?.dispose());

/** Serve a custom /config.js, the way an operator edits it on the static host. */
async function useConfig(page: Page, config: Record<string, string>) {
  await page.route('**/config.js', (route) =>
    route.fulfill({ contentType: 'text/javascript', body: `window.__SPARROW_CONFIG__ = ${JSON.stringify(config)};` }),
  );
}

async function seedWebhook() {
  const consumer = newConsumer();
  const url = `https://example.com/pw-split-${Date.now()}`;
  await registerWebhook(admin, consumer, url);
  return { consumer, url };
}

async function tokenStatus(id: string) {
  const res = await admin.get('/v1/tokens?include_inactive=true');
  expect(res.ok(), await res.text()).toBeTruthy();
  const items = (await res.json()).items as { id: string; status: string }[];
  return items.find((t) => t.id === id)?.status;
}

async function createInvite(name: string, opts: { consumer?: string; ttl?: number } = {}) {
  const res = await admin.post('/v1/invites', { data: { name, consumer: opts.consumer, ttl_seconds: opts.ttl } });
  expect(res.status(), await res.text()).toBe(201);
  return (await res.json()) as { secret: string; path: string; invite: { id: string } };
}

async function expectWebhookListed(page: Page, url: string, consumer: string) {
  await page.getByPlaceholder('Search URL, description, or ID…').fill(url);
  await expect(page.getByRole('row').filter({ hasText: consumer })).toContainText('pw webhook');
}

test('the server rejects requests without the key', async () => {
  const anon = await playwrightRequest.newContext({ baseURL: API });
  expect((await anon.get('/v1/webhooks')).status()).toBe(401);
  await anon.dispose();
});

test('standalone UI without a key asks for one and stores a browser token, never the master key', async ({ page }) => {
  const { consumer, url } = await seedWebhook();
  await useConfig(page, { apiUrl: API });

  await page.goto(`${UI}/webhooks`);
  const prompt = page.getByRole('dialog', { name: 'Sign in to Sparrow' });
  await expect(prompt).toBeVisible();
  await expect(prompt).toContainText('requires an API key');

  await prompt.getByLabel('API key or access token').fill(KEY);
  await prompt.getByRole('button', { name: 'Sign in' }).click();

  await expect(page.getByRole('heading', { level: 1, name: 'Webhooks' })).toBeVisible();
  await expectWebhookListed(page, url, consumer);
  await expect(prompt).toBeHidden();
  const stored = await page.evaluate(() => localStorage.getItem('sparrow_api_key'));
  expect(stored).toMatch(/^sparrow_tk_/);
  expect(stored).not.toBe(KEY);
  await expect(page.getByTestId('account-badge')).toContainText('Signed in as Browser sign-in (Chrome');

  // Survives a reload without asking again.
  await page.reload();
  await expectWebhookListed(page, url, consumer);
  await expect(prompt).toBeHidden();

  // Signing out revokes this browser's token on the server, not just locally.
  const tokenId = await page.evaluate(() => localStorage.getItem('sparrow_api_key_token_id'));
  await page.getByRole('button', { name: 'Sign out' }).click();
  await expect(page.getByRole('dialog', { name: 'Sign in to Sparrow' })).toBeVisible();
  expect(await page.evaluate(() => localStorage.getItem('sparrow_api_key'))).toBeNull();
  expect((await tokenStatus(tokenId!))).toBe('revoked');
});

test('a wrong key is reported as rejected and can be replaced', async ({ page }) => {
  const { consumer, url } = await seedWebhook();
  await useConfig(page, { apiUrl: API });
  // Seed a stale key on the first load only (init scripts run on every navigation).
  await page.addInitScript(() => {
    if (sessionStorage.getItem('pw-seeded')) return;
    sessionStorage.setItem('pw-seeded', '1');
    localStorage.setItem('sparrow_api_key', 'not-the-key');
  });

  await page.goto(`${UI}/webhooks`);
  const prompt = page.getByRole('dialog', { name: 'Sign in to Sparrow' });
  await expect(prompt).toContainText('rejected the current key');

  // A wrong key typed into the prompt is reported inline, without reloading.
  await prompt.getByLabel('API key or access token').fill('still-wrong');
  await prompt.getByRole('button', { name: 'Sign in' }).click();
  await expect(prompt.getByRole('alert')).toContainText('rejected');

  await prompt.getByLabel('API key or access token').fill(KEY);
  await prompt.getByRole('button', { name: 'Sign in' }).click();
  await expectWebhookListed(page, url, consumer);
});

test('apiKey set in config.js is used without prompting', async ({ page }) => {
  const { consumer, url } = await seedWebhook();
  await useConfig(page, { apiUrl: API, apiKey: KEY });

  const apiCalls: string[] = [];
  page.on('request', (r) => r.url().includes('/v1/') && apiCalls.push(r.url()));

  await page.goto(`${UI}/webhooks`);
  await expectWebhookListed(page, url, consumer);
  await expect(page.getByRole('dialog', { name: 'Sign in to Sparrow' })).toHaveCount(0);

  // Every API call went to the API origin, none to the static host.
  expect(apiCalls.length).toBeGreaterThan(0);
  for (const u of apiCalls) expect(u.startsWith(`${API}/v1/`), u).toBeTruthy();
});

test('server links (API docs) point at the API host, not the UI host', async ({ page }) => {
  await useConfig(page, { apiUrl: API, apiKey: KEY });
  await page.goto(`${UI}/webhooks`);
  await expect(page.getByRole('link', { name: 'Docs', exact: true })).toHaveAttribute('href', `${API}/docs`);
});

test('the shipped config.js is a harmless default', async ({ page }) => {
  const res = await page.request.get(`${UI}/config.js`);
  expect(res.ok()).toBeTruthy();
  const body = await res.text();
  expect(body).toContain('window.__SPARROW_CONFIG__');
  expect(body).not.toMatch(/^\s*apiKey:/m);
});

test('consumer portal works from the standalone UI through the API gateway', async ({ page }) => {
  const consumer = newConsumer();
  const url = `https://example.com/pw-split-portal-${Date.now()}`;
  await registerWebhook(admin, consumer, url);
  const { token } = await mintPortalToken(admin, consumer);
  await useConfig(page, { apiUrl: API });

  const calls: string[] = [];
  page.on('request', (r) => r.url().includes('/api/') && calls.push(r.url()));

  await page.goto(`${UI}/portal#token=${token}`);
  await expect(page.getByRole('heading', { name: 'Your Endpoints' })).toBeVisible();
  await expect(page.getByRole('button', { name: new RegExp(url) })).toBeVisible();
  await expect(page.getByRole('dialog', { name: 'Sign in to Sparrow' })).toHaveCount(0);

  expect(calls.length).toBeGreaterThan(0);
  for (const u of calls) expect(u.startsWith(`${API}/portal/api/`), u).toBeTruthy();
});

test('the embedded UI on the same server never receives the key: it asks to sign in', async ({ page }) => {
  const { consumer, url } = await seedWebhook();
  const html = await (await page.request.get(`${API}/webhooks`)).text();
  expect(html).not.toContain(KEY);
  expect(html).not.toContain('apiKey');

  await page.goto(`${API}/webhooks`);
  const prompt = page.getByRole('dialog', { name: 'Sign in to Sparrow' });
  await prompt.getByLabel('API key or access token').fill(KEY);
  await prompt.getByRole('button', { name: 'Sign in' }).click();
  await expectWebhookListed(page, url, consumer);
  await expect(prompt).toHaveCount(0);
});

// ---------------------------------------------------------------------------
// Access tokens and invites (pkg/access): the journeys from the access design.

test('access endpoints need the key; the invite redeem endpoint does not and works once', async () => {
  const anon = await playwrightRequest.newContext({ baseURL: API });
  expect((await anon.post('/v1/tokens', { data: { name: 'x' } })).status()).toBe(401);
  expect((await anon.post('/v1/invites', { data: { name: 'x' } })).status()).toBe(401);
  expect((await anon.get('/v1/tokens')).status()).toBe(401);

  const { secret } = await createInvite(`api-${Date.now()}`);
  const first = await anon.post('/invite/redeem', { data: { invite: secret } });
  expect(first.status()).toBe(200);
  const body = await first.json();
  expect(body.token).toMatch(/^sparrow_tk_/);
  expect(first.headers()['cache-control']).toBe('no-store');

  // The redeemed token works like the master key on /v1...
  const asToken = await playwrightRequest.newContext({ baseURL: API, extraHTTPHeaders: { Authorization: `Bearer ${body.token}` } });
  expect((await asToken.get('/v1/webhooks')).status()).toBe(200);
  const who = await (await asToken.get('/v1/whoami')).json();
  expect(who).toMatchObject({ auth_enabled: true, master_key: false, token_id: body.token_id });
  await asToken.dispose();

  // ...and the invite is spent.
  const again = await anon.post('/invite/redeem', { data: { invite: secret } });
  expect(again.status()).toBe(400);
  expect(await again.text()).not.toContain('sparrow_tk_');
  await anon.dispose();
});

test('an invite link signs a teammate in by name; the same link fails a second time', async ({ page, browser }) => {
  const { consumer, url } = await seedWebhook();
  const name = `alice-${Date.now()}`;
  const { path } = await createInvite(name);
  await useConfig(page, { apiUrl: API });

  await page.goto(`${UI}${path}`);
  await expect(page.getByRole('heading', { level: 1, name: 'Webhooks' })).toBeVisible();
  await expectWebhookListed(page, url, consumer);
  await expect(page.getByTestId('account-badge')).toContainText(`Signed in as ${name}`);
  await expect(page.getByRole('dialog', { name: 'Sign in to Sparrow' })).toHaveCount(0);
  expect(await page.evaluate(() => localStorage.getItem('sparrow_api_key'))).toMatch(/^sparrow_tk_/);
  expect(page.url()).not.toContain('invite=');

  const other = await browser.newContext();
  const page2 = await other.newPage();
  await useConfig(page2, { apiUrl: API });
  await page2.goto(`${UI}${path}`);
  await expect(page2.getByRole('dialog', { name: 'Sign in to Sparrow' })).toContainText('already been used');
  expect(await page2.evaluate(() => localStorage.getItem('sparrow_api_key'))).toBeNull();
  expect(page2.url()).not.toContain('invite=');
  await other.close();
});

test('invite from the Access page, then revoke: the invitee is told their access was revoked', async ({ page, browser }) => {
  await useConfig(page, { apiUrl: API, apiKey: KEY });
  await page.goto(`${UI}/access`);
  await expect(page.getByRole('heading', { level: 1, name: 'Access' })).toBeVisible();

  const name = `bob-${Date.now()}`;
  await page.getByRole('button', { name: 'Invite', exact: true }).click();
  const dialog = page.getByRole('dialog', { name: 'Invite someone' });
  await dialog.getByLabel('Who is it for?').fill(name);
  await dialog.getByRole('button', { name: 'Create invite link' }).click();
  const link = await dialog.getByLabel('Invite link').inputValue();
  expect(link.startsWith(`${UI}/#invite=sparrow_inv_`)).toBeTruthy();
  await dialog.getByRole('button', { name: 'Done' }).click();
  await expect(page.getByTestId('invite-row').filter({ hasText: name })).toBeVisible();

  // Bob opens the link in his own browser.
  const bobCtx = await browser.newContext();
  const bob = await bobCtx.newPage();
  await useConfig(bob, { apiUrl: API });
  await bob.goto(link);
  await expect(bob.getByTestId('account-badge')).toContainText(`Signed in as ${name}`);

  // The invite is no longer pending; Bob's token shows up, created by the master key.
  await page.reload();
  await expect(page.getByTestId('invite-row').filter({ hasText: name })).toHaveCount(0);
  const row = page.getByTestId('token-row').filter({ hasText: name });
  await expect(row).toContainText('Full access');
  await expect(row).toContainText('master key');

  await row.getByRole('button', { name: `Revoke token ${name}` }).click();
  await page.getByRole('dialog', { name: 'Revoke token' }).getByRole('button', { name: 'Revoke' }).click();
  await expect(page.getByTestId('token-row').filter({ hasText: name })).toHaveCount(0);

  // Within the 30 s cache window at worst; this server revokes immediately.
  await bob.reload();
  await expect(bob.getByRole('dialog', { name: 'Sign in to Sparrow' })).toContainText('revoked');
  await bobCtx.close();
});

test('a cancelled invite link cannot be used', async ({ page }) => {
  const name = `carol-${Date.now()}`;
  const { path, invite } = await createInvite(name);
  expect((await admin.delete(`/v1/invites/${invite.id}`)).status()).toBe(204);

  await useConfig(page, { apiUrl: API });
  await page.goto(`${UI}${path}`);
  await expect(page.getByRole('dialog', { name: 'Sign in to Sparrow' })).toContainText('cancelled');
});

test('a token created on the Access page is shown once and works as X-API-Key', async ({ page }) => {
  await useConfig(page, { apiUrl: API, apiKey: KEY });
  await page.goto(`${UI}/access`);
  await page.getByRole('button', { name: 'Create token' }).click();
  const dialog = page.getByRole('dialog', { name: 'Create a token' });
  await dialog.getByLabel('Name').fill(`ci-${Date.now()}`);
  await dialog.getByRole('button', { name: 'Create token' }).click();
  const secret = await dialog.getByLabel('Token').inputValue();
  expect(secret).toMatch(/^sparrow_tk_/);

  const ci = await playwrightRequest.newContext({ baseURL: API, extraHTTPHeaders: { 'X-API-Key': secret } });
  expect((await ci.get('/v1/event-types')).status()).toBe(200);
  await ci.dispose();
});

test('a consumer invite opens that consumer\'s portal and is remembered', async ({ page }) => {
  const consumer = newConsumer();
  const url = `https://example.com/pw-portal-invite-${Date.now()}`;
  await registerWebhook(admin, consumer, url);
  const { path } = await createInvite(`${consumer} staff`, { consumer });
  expect(path.startsWith('/portal#invite=')).toBeTruthy();
  await useConfig(page, { apiUrl: API });

  const calls: string[] = [];
  page.on('request', (r) => r.url().includes('/api/') && calls.push(r.url()));

  await page.goto(`${UI}${path}`);
  await expect(page.getByRole('heading', { name: 'Your Endpoints' })).toBeVisible();
  await expect(page.getByRole('button', { name: new RegExp(url) })).toBeVisible();
  await expect(page).toHaveTitle(new RegExp(consumer));
  expect(page.url()).not.toContain('invite=');
  expect(calls.length).toBeGreaterThan(0);
  for (const u of calls) expect(u.startsWith(`${API}/portal/api/`), u).toBeTruthy();

  // Remembered across reloads and tab restarts (localStorage, not the URL).
  await page.reload();
  await expect(page.getByRole('button', { name: new RegExp(url) })).toBeVisible();
});

test('consumer tokens are pinned to the portal API; full-access tokens cannot use the portal', async () => {
  const consumer = newConsumer();
  await registerWebhook(admin, consumer, `https://example.com/pw-scope-${Date.now()}`);

  const mk = async (body: object) => {
    const res = await admin.post('/v1/tokens', { data: body });
    expect(res.status(), await res.text()).toBe(201);
    return (await res.json()) as { secret: string; token: { expires_at: string | null } };
  };
  const scoped = await mk({ name: 'scoped', consumer });
  expect(scoped.token.expires_at).not.toBeNull(); // consumer tokens always expire
  const full = await mk({ name: 'full' });
  expect(full.token.expires_at).toBeNull(); // tenant-wide tokens never expire by default

  const as = (secret: string) => playwrightRequest.newContext({ baseURL: API, extraHTTPHeaders: { Authorization: `Bearer ${secret}` } });
  const s = await as(scoped.secret);
  expect((await s.get('/v1/webhooks')).status()).toBe(403);
  const hooks = await s.get('/portal/api/webhooks');
  expect(hooks.status()).toBe(200);
  expect(JSON.stringify(await hooks.json())).toContain(consumer);
  expect((await s.post('/portal/api/events', { data: {} })).status()).toBe(403);
  await s.dispose();

  const f = await as(full.secret);
  expect((await f.get('/portal/api/webhooks')).status()).toBe(403);
  await f.dispose();

  // Limits are enforced: consumer tokens live at most 30 days.
  const tooLong = await admin.post('/v1/tokens', { data: { name: 'x', consumer, ttl_seconds: 31 * 86400 } });
  expect(tooLong.status()).toBe(400);
});

test('the embedded UI of a keyed server asks to sign in and never exposes the key', async ({ page }) => {
  const KEYED = process.env.SPARROW_KEYED_URL ?? '';
  test.skip(!KEYED, 'keyed embedded-UI server not running');

  const html = await (await page.request.get(`${KEYED}/webhooks`)).text();
  expect(html).not.toContain(KEY);
  expect(html).not.toContain('apiKey');

  await page.goto(`${KEYED}/webhooks`);
  const prompt = page.getByRole('dialog', { name: 'Sign in to Sparrow' });
  await expect(prompt).toContainText('requires an API key');
  await prompt.getByLabel('API key or access token').fill(KEY);
  await prompt.getByRole('button', { name: 'Sign in' }).click();
  await expect(page.getByRole('heading', { level: 1, name: 'Webhooks' })).toBeVisible();
  await expect(prompt).toBeHidden();
  expect(await page.evaluate(() => localStorage.getItem('sparrow_api_key'))).toMatch(/^sparrow_tk_/);
});
