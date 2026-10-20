import { createServer } from 'node:http';
import type { AddressInfo } from 'node:net';
import { expect, test } from '@playwright/test';
import { ensureAlertDelivery, ensureEventType, mintPortalToken, newConsumer, newEventName, registerWebhook } from './api';

// The consumer portal authenticates purely from the URL fragment (#token=...),
// so every test mints a link via the admin API and visits its path
// (/portal#token=<t>&consumer=<c>&expires=<unix>).

test('portal without a token shows the missing-link empty state', async ({ page }) => {
  await page.goto('/portal');
  await expect(page.getByText('Missing access link')).toBeVisible();
});

test('a valid token opens the portal scoped to its consumer', async ({ page, request }) => {
  const consumer = newConsumer();
  const { path } = await mintPortalToken(request, consumer);
  await page.goto(path);
  await expect(page.getByRole('heading', { name: 'Your Endpoints' })).toBeVisible();
  await expect(page).toHaveTitle(new RegExp(consumer));
  await expect(page.getByRole('link', { name: 'Sparrow' })).toHaveAttribute('href', 'https://github.com/sarathsp06/sparrow');
  await expect(page.getByRole('link', { name: 'Verify webhooks' })).toHaveAttribute('href', 'https://sarathsp06.github.io/sparrow/reference/architecture/#verifying-webhook-signatures');
  await expect(page.getByRole('link', { name: 'llms.txt' })).toHaveAttribute('href', 'https://sarathsp06.github.io/sparrow/llms.txt');
});

test('a consumer can add and pause an endpoint from the portal', async ({ page, request }) => {
  const consumer = newConsumer();
  const { path } = await mintPortalToken(request, consumer);
  const url = `https://example.com/pw-portal-${Date.now()}`;
  const alertEmail = `alerts-${Date.now()}@example.com`;
  // The portal offers the alert email field only when alert delivery is set up.
  await ensureAlertDelivery(request);

  await page.goto(path);
  await page.getByRole('button', { name: 'Add Endpoint' }).click();
  await page.getByLabel('Endpoint URL').fill(url);
  await page.getByLabel(/Health alert email/).fill(alertEmail);
  await page.getByRole('button', { name: 'Add Endpoint', exact: true }).click();
  await expect(page.getByRole('button', { name: new RegExp(url) })).toBeVisible();

  const alerts = await request.get(`/v1/consumers/${encodeURIComponent(consumer)}/alert-configs`);
  expect(alerts.ok(), await alerts.text()).toBeTruthy();
  expect((await alerts.json()).items).toEqual(expect.arrayContaining([
    expect.objectContaining({ email: alertEmail }),
  ]));

  await page.getByRole('button', { name: new RegExp(url) }).click();
  await page.getByRole('button', { name: 'Pause' }).click();
  await expect(page.getByRole('button', { name: 'Resume' }).first()).toBeVisible();
});

test('the theme toggle flips and persists the color scheme', async ({ page, request }) => {
  const { path } = await mintPortalToken(request, newConsumer());
  await page.goto(path);

  const before = await page.evaluate(() => document.documentElement.dataset.theme);
  await page.getByRole('button', { name: 'Toggle color theme' }).click();
  const after = await page.evaluate(() => document.documentElement.dataset.theme);
  expect(after).not.toBe(before);
  expect(await page.evaluate(() => localStorage.getItem('sparrow_portal_theme'))).toBe(after);
});

test('an expired token shows the expired empty state', async ({ page, request }) => {
  const { path } = await mintPortalToken(request, newConsumer(), 1);
  await page.waitForTimeout(1500);
  await page.goto(path);
  await expect(page.getByText('Access link expired')).toBeVisible();
});

test('a bare #token= link (no consumer in the fragment) still reaches the consumer', async ({ page, request }) => {
  const consumer = newConsumer();
  const url = `https://example.com/pw-portal-bare-${Date.now()}`;
  await registerWebhook(request, consumer, url);
  const { token } = await mintPortalToken(request, consumer);
  await page.goto(`/portal#token=${token}`);
  await expect(page.getByRole('button', { name: new RegExp(url) })).toBeVisible();
});

test('the Integrate tab generates a receiver and sends a test event', async ({ page, request }) => {
  // A receiver on this machine that records what Sparrow sends it.
  const received: { headers: Record<string, string | string[] | undefined> }[] = [];
  const receiver = createServer((req, res) => {
    received.push({ headers: req.headers });
    req.resume();
    req.on('end', () => res.writeHead(200, { 'Content-Type': 'application/json' }).end('{"ok":true}'));
  });
  await new Promise<void>((resolve) => receiver.listen(0, '127.0.0.1', resolve));
  try {
    const consumer = newConsumer();
    const event = await ensureEventType(request, newEventName());
    const url = `http://127.0.0.1:${(receiver.address() as AddressInfo).port}/webhook`;
    await registerWebhook(request, consumer, url, [event]);
    const { path } = await mintPortalToken(request, consumer);

    await page.goto(path);
    await page.getByRole('button', { name: new RegExp(url) }).click();
    await page.getByRole('button', { name: 'Integrate' }).click();

    await page.getByRole('button', { name: 'Python (FastAPI)' }).click();
    const files = page.getByRole('navigation', { name: 'Generated files' });
    await expect(files.getByRole('button', { name: 'receiver.py', exact: true })).toBeVisible();
    await expect(page.getByTestId('receiver-code-preview')).toContainText('async def handle_');
    await files.getByRole('button', { name: 'sparrow_verify.py', exact: true }).click();
    await expect(page.getByTestId('receiver-code-preview')).toContainText('def verify_hmac');

    const [download] = await Promise.all([
      page.waitForEvent('download'),
      page.getByRole('button', { name: 'Download .zip' }).click(),
    ]);
    expect(download.suggestedFilename()).toBe('sparrow-receiver-python-fastapi.zip');

    await page.getByTestId('test-event-row').getByRole('button', { name: 'Send test event' }).click();
    await expect(page.getByTestId('test-event-result')).toContainText('✓ 200');
    expect(received).toHaveLength(1);
    expect(received[0].headers['x-sparrow-test']).toBe('true');
    expect(received[0].headers['x-sparrow-event-type']).toBe(event);
  } finally {
    receiver.close();
  }
});
