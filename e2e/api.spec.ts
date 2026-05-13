import { test, expect } from '@playwright/test';

const AUTH = { Authorization: 'Bearer e2e-test-secret' };

test('collect endpoint returns 200', async ({ request }) => {
  const res = await request.get('/api/collect', { headers: AUTH });
  expect(res.status()).toBe(200);
});

test('send endpoint returns 200', async ({ request }) => {
  const res = await request.get('/api/send', { headers: AUTH });
  expect(res.status()).toBe(200);
});
