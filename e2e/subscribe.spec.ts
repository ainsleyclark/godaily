import { test, expect } from '@playwright/test';

// Give the token bucket (1 req/s, burst 10) time to refill between tests.
test.beforeEach(async () => {
  await new Promise((r) => setTimeout(r, 2000));
});

test('subscribe form redirects to thank-you page', async ({ page }) => {
  await page.goto('/');

  const form = page.locator('[data-subscribe]').first();
  await form.locator('input[type="email"]').fill(`sub-${Date.now()}@example.com`);
  await form.locator('button[type="submit"]').click();

  await expect(page).toHaveURL(/\/thank-you\//);
});

test('already subscribed shows inline error', async ({ page }) => {
  const email = `dup-${Date.now()}@example.com`;

  // First subscription via the form to ensure the email is in the DB
  await page.goto('/');
  const form = page.locator('[data-subscribe]').first();
  await form.locator('input[type="email"]').fill(email);
  await form.locator('button[type="submit"]').click();
  await expect(page).toHaveURL(/\/thank-you\//);

  // Second attempt — should stay on the page and show the hint
  await page.goto('/');
  const form2 = page.locator('[data-subscribe]').first();
  await form2.locator('input[type="email"]').fill(email);
  await form2.locator('button[type="submit"]').click();

  const hint = form2.locator('[data-subscribe-hint]');
  await expect(hint).toBeVisible();
  await expect(hint).toContainText("You're already subscribed.");
});

test('invalid email is rejected without navigating away', async ({ page }) => {
  await page.goto('/');

  const form = page.locator('[data-subscribe]').first();
  await form.locator('input[type="email"]').fill('notanemail');
  await form.locator('button[type="submit"]').click();

  // Browser's native email validation blocks the submit event — page stays put
  await expect(page).toHaveURL('/');
});

test('unsubscribe link lands on unsubscribed page', async ({ page }) => {
  const email = `unsub-${Date.now()}@example.com`;

  await page.goto('/');
  const form = page.locator('[data-subscribe]').first();
  await form.locator('input[type="email"]').fill(email);
  await form.locator('button[type="submit"]').click();
  await expect(page).toHaveURL(/\/thank-you\//);

  const res = await page.request.get('/api/e2e/emails');
  const allEmails = await res.json();
  const confirmEmail = allEmails.find(
    (e: { to: string[] }) => Array.isArray(e.to) && e.to.includes(email),
  );
  expect(confirmEmail).toBeTruthy();
  const raw: string = confirmEmail.headers['List-Unsubscribe'].replace(/[<>]/g, '');
  const token = new URL(raw).searchParams.get('token');

  await page.goto(`/api/unsubscribe?token=${token}`);
  await expect(page).toHaveURL('/unsubscribed/');
});

test('re-subscribe after unsubscribe redirects to thank-you', async ({ page }) => {
  const email = `resub-${Date.now()}@example.com`;

  // Subscribe then unsubscribe
  await page.goto('/');
  await page.locator('[data-subscribe]').first().locator('input[type="email"]').fill(email);
  await page.locator('[data-subscribe]').first().locator('button[type="submit"]').click();
  await expect(page).toHaveURL(/\/thank-you\//);

  const res = await page.request.get('/api/e2e/emails');
  const allEmails = await res.json();
  const confirmEmail = allEmails.find(
    (e: { to: string[] }) => Array.isArray(e.to) && e.to.includes(email),
  );
  expect(confirmEmail).toBeTruthy();
  const raw: string = confirmEmail.headers['List-Unsubscribe'].replace(/[<>]/g, '');
  const token = new URL(raw).searchParams.get('token');

  await page.goto(`/api/unsubscribe?token=${token}`);
  await expect(page).toHaveURL('/unsubscribed/');

  // Re-subscribe
  await page.goto('/');
  await page.locator('[data-subscribe]').first().locator('input[type="email"]').fill(email);
  await page.locator('[data-subscribe]').first().locator('button[type="submit"]').click();
  await expect(page).toHaveURL(/\/thank-you\//);
});
