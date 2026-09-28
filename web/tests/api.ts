import { APIRequestContext, expect } from '@playwright/test';

// Admin API helpers. The e2e server runs with no SPARROW_API_KEY, so the REST
// API is open — we drive setup/teardown and JSON-editor flows (event push)
// through it instead of the UI, and assert the UI reflects the result.

const uniq = () => Math.random().toString(36).slice(2, 8);

export const newConsumer = () => `pw-${uniq()}`;
export const newEventName = () => `pw.event.${uniq()}`;

export async function ensureEventType(api: APIRequestContext, name: string, description = 'Playwright event') {
  const res = await api.post('/v1/event-types', { data: { name, description } });
  // 201 created, or 409 if a prior run left it — both fine.
  expect([200, 201, 409]).toContain(res.status());
  return name;
}

export async function registerWebhook(
  api: APIRequestContext,
  consumer: string,
  url: string,
  events: string[] = [],
) {
  const res = await api.post(`/v1/consumers/${encodeURIComponent(consumer)}/webhooks`, {
    data: { url, description: 'pw webhook', active: true, events },
  });
  expect(res.ok(), await res.text()).toBeTruthy();
  return (await res.json()) as { webhook_id: string };
}

export async function pushEvent(api: APIRequestContext, consumer: string, event: string, payload: object = { hello: 'world' }) {
  const res = await api.post(`/v1/consumers/${encodeURIComponent(consumer)}/events?event=${encodeURIComponent(event)}`, {
    data: { payload },
  });
  expect(res.ok(), await res.text()).toBeTruthy();
  return res.json();
}

// A portal link is a consumer-scoped access token (POST /v1/tokens with a consumer).
export async function mintPortalToken(api: APIRequestContext, consumer: string, ttlSeconds?: number) {
  const res = await api.post('/v1/tokens', {
    data: { name: 'pw portal', consumer, ...(ttlSeconds ? { ttl_seconds: ttlSeconds } : {}) },
  });
  expect(res.ok(), await res.text()).toBeTruthy();
  const body = (await res.json()) as { secret: string; portal_path: string; token: { expires_at: string } };
  return { token: body.secret, path: body.portal_path, expires_at: body.token.expires_at };
}
