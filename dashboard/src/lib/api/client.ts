import { dev } from '$app/environment';
import { PUBLIC_API_BASE_URL } from '$env/static/public';
import { getSecret } from '$lib/stores/auth';
import type {
	IssueEngagement,
	ItemMetrics,
	MetricsQuery,
	SourceMetrics,
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
	if (dev) return ''; // use vite proxy
	return PUBLIC_API_BASE_URL || 'https://godaily.dev';
}

function buildUrl(path: string, query?: MetricsQuery): string {
	const url = new URL(`${baseUrl()}${path}`, typeof window === 'undefined' ? 'http://x' : window.location.origin);
	if (query) {
		for (const [k, v] of Object.entries(query)) {
			if (v !== undefined && v !== null && v !== '') url.searchParams.set(k, String(v));
		}
	}
	// keep absolute path-only when in dev (vite proxy)
	if (dev) return `${url.pathname}${url.search}`;
	return url.toString();
}

async function request<T>(
	path: string,
	query?: MetricsQuery,
	overrideSecret?: string
): Promise<T> {
	const secret = overrideSecret ?? getSecret();
	const res = await fetch(buildUrl(path, query), {
		headers: {
			Accept: 'application/json',
			...(secret ? { Authorization: `Bearer ${secret}` } : {})
		}
	});
	if (res.status === 401) {
		if (typeof window !== 'undefined' && !overrideSecret) {
			window.dispatchEvent(new Event('metrics:unauthorized'));
		}
		throw new ApiError(401, 'Unauthorized');
	}
	if (!res.ok) {
		const text = await res.text().catch(() => '');
		throw new ApiError(res.status, text || `HTTP ${res.status}`);
	}
	const body = (await res.json()) as { data: T };
	return body.data;
}

async function login(password: string): Promise<{ token: string }> {
	const res = await fetch(buildUrl('/api/auth/login'), {
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
		request<SummaryStats>('/api/metrics/summary', q, secret),
	issues: (q?: MetricsQuery) => request<IssueEngagement[]>('/api/metrics/issues', q),
	items: (q?: MetricsQuery) => request<ItemMetrics[]>('/api/metrics/items', q),
	tags: (q?: MetricsQuery) => request<TagMetrics[]>('/api/metrics/tags', q),
	sources: (q?: MetricsQuery) => request<SourceMetrics[]>('/api/metrics/sources', q),
	trend: (q?: MetricsQuery) => request<TrendData>('/api/metrics/trend', q),
	subscribers: (q?: MetricsQuery) => request<SubscriberData>('/api/metrics/subscribers', q)
};
