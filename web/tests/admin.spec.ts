import { expect, test } from '@playwright/test';
import { ensureEventType, newConsumer, newEventName, pushEvent, registerWebhook } from './api';

// The operator console defaults to the "all consumers" scope, so data seeded
// under any consumer via the open API shows up in these global list pages.

test('home redirects to the webhooks list', async ({ page }) => {
  await page.goto('/');
  await expect(page).toHaveURL(/\/webhooks\/?$/);
  await expect(page.getByRole('heading', { level: 1, name: 'Webhooks' })).toBeVisible();
  await expect(page.getByRole('link', { name: 'Star Sparrow on GitHub' })).toHaveAttribute('href', 'https://github.com/sarathsp06/sparrow');
});

test('core pages render without uncaught errors', async ({ page }) => {
  const crashes: string[] = [];
  page.on('pageerror', (e) => crashes.push(String(e)));
  const pages: [string, string][] = [
    ['/webhooks', 'Webhooks'],
    ['/events', 'Events'],
    ['/deliveries', 'Deliveries'],
    ['/dashboard/health', 'Health Dashboard'],
  ];
  for (const [path, heading] of pages) {
    await page.goto(path);
    await expect(page.getByRole('heading', { level: 1, name: heading })).toBeVisible();
  }
  expect(crashes, crashes.join('\n')).toHaveLength(0);
});

test('register webhook hides recipe choices until opened', async ({ page }) => {
  await page.goto('/webhooks/register');
  await expect(page.getByLabel(/Health alert email/)).toBeVisible();
  await expect(page.getByRole('button', { name: /Clickhouse/i })).toHaveCount(0);

  await page.getByRole('button', { name: /Recipes/ }).click();
  await expect(page.getByRole('button', { name: /Clickhouse/i })).toBeVisible();

  await page.getByRole('button', { name: /Clickhouse/i }).click();
  await expect(page.getByRole('button', { name: 'Use recipe' })).toBeVisible();
});

test('registering an event type through the UI adds it to the catalog', async ({ page }) => {
  const name = newEventName();
  await page.goto('/events/register');
  await page.getByLabel('Event Name').fill(name);
  await page.getByLabel('Description').fill('Registered by Playwright');
  await page.getByRole('button', { name: 'Register Event' }).click();

  await expect(page).toHaveURL(/\/events\/?$/);
  await page.getByPlaceholder('Search by name or description…').fill(name);
  await expect(page.getByText(name, { exact: true })).toBeVisible();
});

test('a webhook created via the API appears in the operator list', async ({ page, request }) => {
  const consumer = newConsumer();
  const url = `https://example.com/pw-${Date.now()}`;
  await registerWebhook(request, consumer, url);

  await page.goto('/webhooks');
  await page.getByPlaceholder('Search URL, description, or ID…').fill(url);
  await expect(page.getByRole('row').filter({ hasText: consumer })).toContainText('pw webhook');
  await expect(page.getByText(consumer, { exact: false })).toBeVisible();
});

test('pushed events produce delivery attempts visible in the console', async ({ page, request }) => {
  const consumer = newConsumer();
  const event = newEventName();
  await ensureEventType(request, event);
  const { webhook_id } = await registerWebhook(request, consumer, `https://example.com/pw-del-${Date.now()}`, [event]);
  await pushEvent(request, consumer, event);

  await page.goto('/deliveries');
  await page.getByPlaceholder('Webhook ID').fill(webhook_id);
  // Delivery is queued asynchronously by River; poll until the row lands.
  await expect(async () => {
    await page.reload();
    await page.getByPlaceholder('Webhook ID').fill(webhook_id);
    await expect(page.getByText(webhook_id.slice(0, 8))).toBeVisible({ timeout: 2000 });
  }).toPass({ timeout: 20000 });
});

test('health dashboard renders fleet statistics', async ({ page }) => {
  await page.goto('/dashboard/health');
  await expect(page.getByRole('heading', { level: 1, name: 'Health Dashboard' })).toBeVisible();
  await expect(page.getByText('Total Webhooks')).toBeVisible();
});
