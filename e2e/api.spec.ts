import { test, expect } from '@playwright/test';

test('collect endpoint returns 200', async ({ request }) => {
  const res = await request.get('/api/collect');
  expect(res.status()).toBe(200);
});

test('send endpoint returns 200', async ({ request }) => {
  const res = await request.get('/api/send');
  expect(res.status()).toBe(200);
});
