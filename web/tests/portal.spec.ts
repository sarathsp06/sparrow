import { expect, test } from '@playwright/test';
import { mintPortalToken, newConsumer } from './api';

// The consumer portal authenticates purely from the URL fragment (#token=...),
// so every test mints a token via the admin API and visits /portal#token=<t>.

test('portal without a token shows the missing-link empty state', async ({ page }) => {
  await page.goto('/portal');
  await expect(page.getByText('Missing access link')).toBeVisible();
});

test('a valid token opens the portal scoped to its consumer', async ({ page, request }) => {
  const consumer = newConsumer();
  const { token } = await mintPortalToken(request, consumer);
  await page.goto(`/portal#token=${token}`);
  await expect(page.getByRole('heading', { name: 'Your Endpoints' })).toBeVisible();
  await expect(page).toHaveTitle(new RegExp(consumer));
  await expect(page.getByRole('link', { name: 'Sparrow' })).toHaveAttribute('href', 'https://github.com/sarathsp06/sparrow');
  await expect(page.getByRole('link', { name: 'Verify webhooks' })).toHaveAttribute('href', 'https://sarathsp06.github.io/sparrow/reference/architecture/#verifying-webhook-signatures');
  await expect(page.getByRole('link', { name: 'llms.txt' })).toHaveAttribute('href', 'https://sarathsp06.github.io/sparrow/llms.txt');
});

test('a consumer can add and pause an endpoint from the portal', async ({ page, request }) => {
  const consumer = newConsumer();
  const { token } = await mintPortalToken(request, consumer);
  const url = `https://example.com/pw-portal-${Date.now()}`;
  const alertEmail = `alerts-${Date.now()}@example.com`;

  await page.goto(`/portal#token=${token}`);
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
  const { token } = await mintPortalToken(request, newConsumer());
  await page.goto(`/portal#token=${token}`);

  const before = await page.evaluate(() => document.documentElement.dataset.theme);
  await page.getByRole('button', { name: 'Toggle color theme' }).click();
  const after = await page.evaluate(() => document.documentElement.dataset.theme);
  expect(after).not.toBe(before);
  expect(await page.evaluate(() => localStorage.getItem('sparrow_portal_theme'))).toBe(after);
});

test('an expired token shows the expired empty state', async ({ page, request }) => {
  const { token } = await mintPortalToken(request, newConsumer(), 1);
  await page.waitForTimeout(1500);
  await page.goto(`/portal#token=${token}`);
  await expect(page.getByText('Access link expired')).toBeVisible();
});
