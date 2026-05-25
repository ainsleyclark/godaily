import { test, expect } from '@playwright/test';

const NAMESPACE = process.env.TESTMAIL_NAMESPACE!;
const API_KEY = process.env.TESTMAIL_API_KEY!;
const AUTH = { Authorization: `Bearer ${process.env.API_SECRET}` };
const tag = `e2e-${Date.now()}`;
const email = `${NAMESPACE}.${tag}@inbox.testmail.app`;

async function waitForEmail(
  subject: string,
): Promise<{ subject: string; html: string }> {
  const url = `https://api.testmail.app/api/json?apikey=${API_KEY}&namespace=${NAMESPACE}&tag=${tag}&livequery=true&timeout=20000`;
  const res = await fetch(url);
  const data = (await res.json()) as { emails?: Array<{ subject: string; html: string }> };
  const msg = data.emails?.find((m) => m.subject.includes(subject));
  if (!msg) throw new Error(`Email with subject containing "${subject}" not found`);
  return msg;
}

test.describe.serial('full production pipeline', () => {
  test('subscribe form redirects to thank-you', async ({ page }) => {
    await page.goto('/');
    await page
      .locator('[data-subscribe]')
      .first()
      .locator('input[type="email"]')
      .fill(email);
    await page
      .locator('[data-subscribe]')
      .first()
      .locator('button[type="submit"]')
      .click();
    await expect(page).toHaveURL(/\/thank-you\//);
  });

  test('confirmation email arrives and subscriber is confirmed', async ({ request }) => {
    const msg = await waitForEmail('Confirm');
    const confirmLink = msg.html.match(/href="([^"]*\/api\/confirm[^"]*)"/)?.[1];
    expect(confirmLink, 'confirmation link not found in email').toBeTruthy();
    const token = new URL(confirmLink!).searchParams.get('token');
    const res = await request.get(`/api/confirm?token=${token}`);
    expect(res.status()).toBe(200);
  });

  test('collect fetches items with no source errors', async ({ request }) => {
    const res = await request.get('/api/collect?force=true', { headers: AUTH });
    expect(res.status()).toBe(200);
    const body = (await res.json()) as {
      sources: Record<string, { count: number; error: string | null }>;
    };
    for (const [src, result] of Object.entries(body.sources)) {
      expect(result.error, `source ${src} failed`).toBeNull();
    }
    const total = Object.values(body.sources).reduce((sum, r) => sum + r.count, 0);
    expect(total).toBeGreaterThan(0);
  });

  test('build creates a draft issue', async ({ request }) => {
    const res = await request.get('/api/build?force=true', { headers: AUTH });
    expect(res.status()).toBe(200);
    const issues = (await (
      await request.get('/api/issues?status=draft', { headers: AUTH })
    ).json()) as { data: Array<{ status: string }> };
    expect(issues.data.some((i) => i.status === 'draft')).toBe(true);
  });

  test('send delivers digest to subscriber', async ({ request }) => {
    const res = await request.get('/api/send?force=true', { headers: AUTH });
    expect(res.status()).toBe(200);
    const digest = await waitForEmail('GoDaily');
    expect(digest.html.includes('href=')).toBe(true);
    const issues = (await (
      await request.get('/api/issues', { headers: AUTH })
    ).json()) as { data: Array<{ status: string }> };
    expect(issues.data[0].status).toBe('sent');
  });
});
