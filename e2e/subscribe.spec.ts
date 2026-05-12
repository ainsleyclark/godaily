import { test, expect } from '@playwright/test';

test('subscribe form redirects to thank-you page', async ({ page }) => {
  await page.goto('/');

  const form = page.locator('[data-subscribe]').first();
  await form.locator('input[type="email"]').fill(`sub-${Date.now()}@example.com`);
  await form.locator('button[type="submit"]').click();

  await expect(page).toHaveURL('/thank-you/');
});

test('already subscribed shows inline error', async ({ page }) => {
  const email = `dup-${Date.now()}@example.com`;

  // First subscription via the form to ensure the email is in the DB
  await page.goto('/');
  const form = page.locator('[data-subscribe]').first();
  await form.locator('input[type="email"]').fill(email);
  await form.locator('button[type="submit"]').click();
  await expect(page).toHaveURL('/thank-you/');

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
