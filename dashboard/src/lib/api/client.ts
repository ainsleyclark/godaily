import { dev } from '$app/environment';
import { PUBLIC_API_BASE_URL } from '$env/static/public';
import { getSecret } from '$lib/stores/auth';
import createClient, { type Middleware } from 'openapi-fetch';
import type { paths } from './schema';
import type {
	DigestIssue,
	IssueEngagement,
	IssueStatus,
	ItemMetrics,
	MetricsQuery,
	PaginatedResponse,
	SocialPostMetric,
	SourceMetrics,
	Subscriber,
	SubscriberData,
	SummaryStats,
	TagMetrics,
	TrendData
} from './types';

export class ApiError extends Error {
	status: number;
	constructor(status: number, message: string) {
		super(message);
		this.status = status;
	}
}

function baseUrl(): string {
	// The OpenAPI contract documents paths under the /api base path, but swaggo
	// emits an empty server URL so the generated `paths` keys omit it — fold
	// /api into the client base. In dev we stay origin-relative so Vite's /api
	// proxy forwards the request.
	if (dev) return '/api';
	return `${PUBLIC_API_BASE_URL || 'https://godaily.dev'}/api`;
}

const middleware: Middleware = {
	onRequest({ request }) {
		// Vercel's trailingSlash:true would otherwise 308-redirect the request,
		// which browsers reject on CORS preflights — land on the canonical URL.
		const url = new URL(request.url);
		if (!url.pathname.endsWith('/')) {
			url.pathname += '/';
			request = new Request(url, request);
		}
		request.headers.set('Accept', 'application/json');
		// Don't clobber a per-call Authorization header (used to validate a
		// freshly entered secret before it's stored).
		if (!request.headers.has('Authorization')) {
			const secret = getSecret();
			if (secret) request.headers.set('Authorization', `Bearer ${secret}`);
		}
		return request;
	},
	onResponse({ response }) {
		if (response.status === 401 && typeof window !== 'undefined') {
			window.dispatchEvent(new Event('metrics:unauthorized'));
		}
		return response;
	}
};

const client = createClient<paths>({ baseUrl: baseUrl() });
client.use(middleware);

type FetchResult = { data?: unknown; error?: unknown; response: Response };

// Unwraps the `{ data, error, message }` API envelope, surfacing the inner
// payload and converting non-2xx responses into ApiError.
async function unwrap<T>(p: Promise<FetchResult>): Promise<T> {
	const { data, response } = await p;
	if (!response.ok) {
		if (response.status === 401) throw new ApiError(401, 'Unauthorized');
		const body = await response
			.clone()
			.json()
			.catch(() => null);
		const message =
			(body && typeof body === 'object' && 'message' in body && typeof body.message === 'string'
				? body.message
				: '') || `HTTP ${response.status}`;
		throw new ApiError(response.status, message);
	}
	return (data as { data: T }).data;
}

async function login(password: string): Promise<{ token: string }> {
	// Hits the dashboard's own SvelteKit server endpoint (same origin), not the
	// Go API — so the password and API secret stay server-side.
	const res = await fetch('/auth/login', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
		body: JSON.stringify({ password })
	});
	if (res.status === 401) throw new ApiError(401, 'Unauthorized');
	if (!res.ok) {
		const text = await res.text().catch(() => '');
		throw new ApiError(res.status, text || `HTTP ${res.status}`);
	}
	return ((await res.json()) as { data: { token: string } }).data;
}

export const api = {
	login,
	summary: (q?: MetricsQuery, secret?: string) =>
		unwrap<SummaryStats>(
			client.GET('/metrics/summary', {
				params: { query: { from: q?.from, to: q?.to } },
				...(secret ? { headers: { Authorization: `Bearer ${secret}` } } : {})
			})
		),
	issues: (q?: MetricsQuery) =>
		unwrap<IssueEngagement[]>(
			client.GET('/metrics/issues', {
				params: { query: { from: q?.from, to: q?.to, limit: q?.limit } }
			})
		),
	items: (q?: MetricsQuery) =>
		unwrap<ItemMetrics[]>(
			client.GET('/metrics/items', {
				params: { query: { from: q?.from, to: q?.to, limit: q?.limit } }
			})
		),
	tags: (q?: MetricsQuery) =>
		unwrap<TagMetrics[]>(
			client.GET('/metrics/tags', {
				params: { query: { from: q?.from, to: q?.to, limit: q?.limit } }
			})
		),
	sources: (q?: MetricsQuery) =>
		unwrap<SourceMetrics[]>(
			client.GET('/metrics/sources', {
				params: { query: { from: q?.from, to: q?.to, limit: q?.limit } }
			})
		),
	trend: (q?: MetricsQuery) =>
		unwrap<TrendData>(
			client.GET('/metrics/trend', {
				params: { query: { from: q?.from, to: q?.to, metric: q?.metric, bucket: q?.bucket } }
			})
		),
	subscribers: (q?: MetricsQuery) =>
		unwrap<SubscriberData>(
			client.GET('/metrics/subscribers', {
				params: { query: { from: q?.from, to: q?.to, bucket: q?.bucket } }
			})
		),
	social: (q?: MetricsQuery) =>
		unwrap<SocialPostMetric[]>(
			client.GET('/metrics/social', {
				params: { query: { from: q?.from, to: q?.to } }
			})
		),
	subscriberList: (page = 1, perPage = 50, search = '') =>
		unwrap<PaginatedResponse<Subscriber>>(
			client.GET('/digest/subscribers', {
				params: { query: { page, per_page: perPage, ...(search ? { search } : {}) } }
			})
		),
	digestIssues: (status?: IssueStatus, page = 1, perPage = 100) =>
		unwrap<PaginatedResponse<DigestIssue>>(
			client.GET('/issues', {
				params: { query: { page, per_page: perPage, ...(status ? { status } : {}) } }
			})
		),
	digestIssueById: (id: number) =>
		unwrap<DigestIssue>(
			client.GET('/issues/{key}', { params: { path: { key: String(id) } } })
		),
	updateDigestIssue: (id: number, body: { subject: string; summary: string }) =>
		unwrap<DigestIssue>(client.PATCH('/issues/{id}', { params: { path: { id } }, body })),
	deleteDigestItem: (issueId: number, itemId: number) =>
		unwrap<DigestIssue>(
			client.DELETE('/issues/{id}/items/{itemID}', {
				params: { path: { id: issueId, itemID: itemId } }
			})
		),
	reorderDigestItems: (issueId: number, itemIds: number[]) =>
		unwrap<DigestIssue>(
			client.PATCH('/issues/{id}/items/reorder', {
				params: { path: { id: issueId } },
				body: { item_ids: itemIds }
			})
		)
};
