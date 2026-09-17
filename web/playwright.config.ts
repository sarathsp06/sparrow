import { defineConfig, devices } from '@playwright/test';

// Base URL of a running Sparrow server with SPARROW_SERVE_UI=true.
// scripts/test-ui.sh boots Postgres + the built server and sets this.
const baseURL = process.env.SPARROW_BASE_URL ?? 'http://localhost:8080';

export default defineConfig({
  testDir: './tests',
  fullyParallel: false, // shared server + Postgres; keep data-mutating specs serial
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  workers: 1,
  reporter: process.env.CI ? [['list'], ['html', { open: 'never' }]] : 'list',
  use: {
    baseURL,
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
});
