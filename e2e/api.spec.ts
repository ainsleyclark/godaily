import { test, expect, type APIRequestContext } from '@playwright/test';

const AUTH = { Authorization: 'Bearer e2e-test-secret' };

// ── Existing endpoint smoke tests ─────────────────────────────────────────────

test('collect endpoint returns 200', async ({ request }) => {
  const res = await request.get('/api/collect', { headers: AUTH });
  expect(res.status()).toBe(200);
});

test('build endpoint returns 200', async ({ request }) => {
  const res = await request.get('/api/build', { headers: AUTH });
  expect(res.status()).toBe(200);
});

test('send endpoint returns 200', async ({ request }) => {
  const res = await request.get('/api/send', { headers: AUTH });
  expect(res.status()).toBe(200);
});

// ── Pipeline state assertions ─────────────────────────────────────────────────
//
// These run serially because each step leaves DB state that the next step reads.
// Debug endpoints (/api/e2e/pipeline/*, /api/e2e/db/*) bypass the weekend guard
// and surface raw DB rows so assertions are always deterministic.

test.describe.serial('pipeline state assertions', () => {
  test('collect seeds items in the database', async ({ request }) => {
    const res = await request.post('/api/e2e/pipeline/collect');
    expect(res.status()).toBe(200);

    const { count } = await (await request.get('/api/e2e/db/items/count')).json();
    expect(count).toBeGreaterThan(0);
  });

  test('build creates a draft issue', async ({ request }) => {
    const res = await request.post('/api/e2e/pipeline/build');
    expect(res.status()).toBe(200);

    const dbIssues = await (await request.get('/api/e2e/db/issues')).json();
    expect(dbIssues.length).toBeGreaterThan(0);
    expect(dbIssues.some((i: { status: string }) => i.status === 'draft')).toBe(true);
  });

  test('send delivers email to subscribers and marks issue as sent', async ({ request }) => {
    // Subscribe and confirm a test subscriber so SendDigest has someone to send to.
    await request.post('/api/subscribe', { data: { email: 'pipeline-send@e2e.test' } });
    const allEmails = await (await request.get('/api/e2e/emails')).json();
    const confirmEmail = allEmails.find(
      (e: { to: string[] }) => Array.isArray(e.to) && e.to.includes('pipeline-send@e2e.test'),
    );
    expect(confirmEmail).toBeTruthy();
    const tokenMatch = (confirmEmail.text as string).match(/token=([^\s]+)/);
    expect(tokenMatch).toBeTruthy();
    await request.get(`/api/confirm?token=${tokenMatch![1]}`);

    const sendRes = await request.post('/api/e2e/pipeline/send');
    expect(sendRes.status()).toBe(200);

    // At least one digest email (admin + subscriber) must have been captured.
    const sentEmails = await (await request.get('/api/e2e/emails')).json();
    expect(
      sentEmails.some((e: { subject: string }) => e.subject.includes('GoDaily')),
    ).toBe(true);

    // Public issues API must now return the issue as 'sent'.
    const issuesRes = await request.get('/api/issues', { headers: AUTH });
    expect(issuesRes.status()).toBe(200);
    const issuesData = await issuesRes.json();
    expect(issuesData.total).toBeGreaterThan(0);
    expect(issuesData.data[0].status).toBe('sent');
  });
});

// ── Resend webhook tests ──────────────────────────────────────────────────────
//
// /api/e2e/sign generates valid Svix HMAC signatures using the test secret so
// we can POST authentic-looking webhook payloads without real Resend credentials.

async function signedHeaders(
  request: APIRequestContext,
  body: string,
  id?: string,
): Promise<Record<string, string>> {
  const msgID = id ?? `msg_${Date.now()}_${Math.random().toString(36).slice(2, 9)}`;
  const timestamp = Math.floor(Date.now() / 1000).toString();
  const signRes = await request.post('/api/e2e/sign', {
    data: { body, id: msgID, timestamp },
  });
  const signed = await signRes.json();
  return {
    'svix-id': signed['svix-id'],
    'svix-timestamp': signed['svix-timestamp'],
    'svix-signature': signed['svix-signature'],
    'Content-Type': 'application/json',
  };
}

test.describe('resend webhooks', () => {
  test('wrong HTTP method returns 405', async ({ request }) => {
    const res = await request.get('/api/webhooks/resend');
    expect(res.status()).toBe(405);
  });

  test('invalid signature returns 401', async ({ request }) => {
    const body = JSON.stringify({
      type: 'email.delivered',
      created_at: '2026-01-01T00:00:00Z',
      data: { email_id: 'bad-sig', to: ['x@example.com'], subject: 'GoDaily', tags: {} },
    });
    const res = await request.post('/api/webhooks/resend', {
      data: body,
      headers: {
        'svix-id': 'msg_tampered',
        'svix-timestamp': '1234567890',
        'svix-signature': 'v1,invalidsignature==',
        'Content-Type': 'application/json',
      },
    });
    expect(res.status()).toBe(401);
  });

  test('untracked event type (email.sent) returns 200', async ({ request }) => {
    const body = JSON.stringify({
      type: 'email.sent',
      created_at: '2026-01-01T00:00:00Z',
      data: { email_id: 'untracked-1', to: ['x@example.com'], subject: 'GoDaily', tags: {} },
    });
    const headers = await signedHeaders(request, body);
    const res = await request.post('/api/webhooks/resend', { data: body, headers });
    expect(res.status()).toBe(200);
  });

  test('valid delivered event is stored and returns 200', async ({ request }) => {
    const eventID = `msg_delivered_${Date.now()}`;
    const body = JSON.stringify({
      type: 'email.delivered',
      created_at: '2026-01-01T00:00:00Z',
      data: { email_id: 'delivered-1', to: ['delivered@example.com'], subject: 'GoDaily', tags: {} },
    });
    const headers = await signedHeaders(request, body, eventID);

    const { count: before } = await (await request.get('/api/e2e/db/events/count')).json();
    const res = await request.post('/api/webhooks/resend', { data: body, headers });
    expect(res.status()).toBe(200);
    const { count: after } = await (await request.get('/api/e2e/db/events/count')).json();
    expect(after).toBe(before + 1);
  });

  test('bounced event marks subscriber as bounced', async ({ request }) => {
    const bounceEmail = `bounce-${Date.now()}@e2e.test`;
    await request.post('/api/subscribe', { data: { email: bounceEmail } });

    const body = JSON.stringify({
      type: 'email.bounced',
      created_at: '2026-01-01T00:00:00Z',
      data: {
        email_id: `bounce-evt-${Date.now()}`,
        to: [bounceEmail],
        subject: 'GoDaily',
        tags: {},
      },
    });
    const headers = await signedHeaders(request, body);
    const res = await request.post('/api/webhooks/resend', { data: body, headers });
    expect(res.status()).toBe(200);

    const subs = await (await request.get('/api/e2e/db/subscribers')).json();
    const sub = subs.find((s: { email: string }) => s.email === bounceEmail);
    expect(sub).toBeTruthy();
    expect(sub.bounced_at).not.toBe('');
  });

  test('duplicate event is idempotent', async ({ request }) => {
    const dedupID = `msg_dedup_${Date.now()}`;
    const body = JSON.stringify({
      type: 'email.delivered',
      created_at: '2026-01-01T00:00:00Z',
      data: { email_id: 'dedup-1', to: ['dedup@example.com'], subject: 'GoDaily', tags: {} },
    });
    // Both requests must carry the same svix-id so EventID matches in the DB.
    const headers = await signedHeaders(request, body, dedupID);

    const { count: before } = await (await request.get('/api/e2e/db/events/count')).json();
    const res1 = await request.post('/api/webhooks/resend', { data: body, headers });
    const res2 = await request.post('/api/webhooks/resend', { data: body, headers });
    expect(res1.status()).toBe(200);
    expect(res2.status()).toBe(200);
    const { count: after } = await (await request.get('/api/e2e/db/events/count')).json();
    expect(after).toBe(before + 1);
  });
});
