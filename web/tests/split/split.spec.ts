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

async function expectWebhookListed(page: Page, url: string, consumer: string) {
  await page.getByPlaceholder('Search URL, description, or ID…').fill(url);
  await expect(page.getByRole('row').filter({ hasText: consumer })).toContainText('pw webhook');
}

test('the server rejects requests without the key', async () => {
  const anon = await playwrightRequest.newContext({ baseURL: API });
  expect((await anon.get('/v1/webhooks')).status()).toBe(401);
  await anon.dispose();
});

test('standalone UI without a key asks for one, remembers it, and can forget it', async ({ page }) => {
  const { consumer, url } = await seedWebhook();
  await useConfig(page, { apiUrl: API });

  await page.goto(`${UI}/webhooks`);
  const prompt = page.getByRole('dialog', { name: 'API key required' });
  await expect(prompt).toBeVisible();
  await expect(prompt).toContainText('requires an API key');

  await prompt.getByLabel('API key').fill(KEY);
  await prompt.getByRole('button', { name: 'Save key' }).click();

  await expect(page.getByRole('heading', { level: 1, name: 'Webhooks' })).toBeVisible();
  await expectWebhookListed(page, url, consumer);
  await expect(prompt).toBeHidden();
  expect(await page.evaluate(() => localStorage.getItem('sparrow_api_key'))).toBe(KEY);

  // Survives a reload without asking again.
  await page.reload();
  await expectWebhookListed(page, url, consumer);
  await expect(prompt).toBeHidden();

  await page.getByRole('button', { name: 'Forget API key' }).click();
  await expect(page.getByRole('dialog', { name: 'API key required' })).toBeVisible();
  expect(await page.evaluate(() => localStorage.getItem('sparrow_api_key'))).toBeNull();
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
  const prompt = page.getByRole('dialog', { name: 'API key required' });
  await expect(prompt).toContainText('rejected the current API key');

  await prompt.getByLabel('API key').fill(KEY);
  await prompt.getByRole('button', { name: 'Save key' }).click();
  await expectWebhookListed(page, url, consumer);
});

test('apiKey set in config.js is used without prompting', async ({ page }) => {
  const { consumer, url } = await seedWebhook();
  await useConfig(page, { apiUrl: API, apiKey: KEY });

  const apiCalls: string[] = [];
  page.on('request', (r) => r.url().includes('/v1/') && apiCalls.push(r.url()));

  await page.goto(`${UI}/webhooks`);
  await expectWebhookListed(page, url, consumer);
  await expect(page.getByRole('dialog', { name: 'API key required' })).toHaveCount(0);

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
  await expect(page.getByRole('dialog', { name: 'API key required' })).toHaveCount(0);

  expect(calls.length).toBeGreaterThan(0);
  for (const u of calls) expect(u.startsWith(`${API}/portal/api/`), u).toBeTruthy();
});

test('the embedded UI on the same server gets the key injected (no prompt)', async ({ page }) => {
  const { consumer, url } = await seedWebhook();
  await page.goto(`${API}/webhooks`);
  await expectWebhookListed(page, url, consumer);
  await expect(page.getByRole('dialog', { name: 'API key required' })).toHaveCount(0);
});
