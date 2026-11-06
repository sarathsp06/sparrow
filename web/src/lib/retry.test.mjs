import assert from "node:assert/strict";
import test from "node:test";

import { describeRetryWindow, retryWindowSeconds, DEFAULT_MAX_RETRIES, DEFAULT_RETRY_BACKOFF_SECONDS } from "./retry.ts";

test("default schedule retries for about three days", () => {
  // 1+2+...+1024 minutes, then two 24 h delays.
  assert.equal(retryWindowSeconds(DEFAULT_MAX_RETRIES, DEFAULT_RETRY_BACKOFF_SECONDS), 2047 * 60 + 2 * 86400);
  assert.equal(describeRetryWindow(DEFAULT_MAX_RETRIES, DEFAULT_RETRY_BACKOFF_SECONDS), "about 3.4 days");
});

test("old default of three retries spans seven minutes", () => {
  assert.equal(retryWindowSeconds(3, 60), 7 * 60);
  assert.equal(describeRetryWindow(3, 60), "about 7 minutes");
});

test("delays are capped at 24 hours", () => {
  assert.equal(retryWindowSeconds(2, 3600 * 24), 2 * 86400);
});

test("no retries", () => {
  assert.equal(describeRetryWindow(0, 60), "no retries");
  assert.equal(describeRetryWindow(3, 0), "no retries");
});
