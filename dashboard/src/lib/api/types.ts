// Public type aliases for the dashboard, derived from the OpenAPI contract.
//
// `schema.d.ts` is generated from `docs/openapi/swagger.yaml` (run `pnpm gen:api`).
// Treat it as the source of truth and re-export friendly names here rather than
// importing the verbose `components['schemas']['package.Type']` keys directly.
import type { components } from './schema';

type Schemas = components['schemas'];

// swaggo/swag does not emit `required` markers, so every property in the
// generated schema is optional. The API always populates these response
// fields, so we assert their presence here to keep call sites ergonomic. If a
// field is genuinely optional, model it on the Go struct (e.g. omitempty) and
// document it — don't loosen things by hand in this file.
type DeepRequired<T> = T extends (infer U)[]
	? DeepRequired<U>[]
	: T extends object
		? { [K in keyof T]-?: DeepRequired<T[K]> }
		: T;

export type SummaryStats = DeepRequired<Schemas['engagement.SummaryStats']>;
export type IssueEngagement = DeepRequired<Schemas['engagement.IssueEngagement']>;
export type ItemMetrics = DeepRequired<Schemas['engagement.ItemMetrics']>;
export type TagMetrics = DeepRequired<Schemas['engagement.TagMetrics']>;
export type SourceMetrics = DeepRequired<Schemas['engagement.SourceMetrics']>;
export type TrendPoint = DeepRequired<Schemas['engagement.TrendPoint']>;
export type TrendData = DeepRequired<Schemas['engagement.TrendData']>;
export type SubscriberPoint = DeepRequired<Schemas['engagement.SubscriberPoint']>;
export type SubscriberData = DeepRequired<Schemas['engagement.SubscriberData']>;
export type SocialPostMetric = DeepRequired<Schemas['engagement.SocialPostEngagement']>;
export type IssueStats = DeepRequired<Schemas['engagement.IssueStats']>;
export type LinkClicks = DeepRequired<Schemas['engagement.LinkClicks']>;
export type IssueDetail = DeepRequired<Schemas['metrics.IssueDetail']>;
export type Subscriber = DeepRequired<Schemas['audience.Subscriber']>;

export type IssueStatus = Schemas['digest.IssueStatus'];
export type DigestAuthor = DeepRequired<Schemas['news.Author']>;
export type DigestItem = DeepRequired<Schemas['news.Item']>;
export type DigestIssue = DeepRequired<Schemas['digest.Issue']>;

export type SocialPost = DeepRequired<Schemas['social.Post']>;
export type SocialPostKind = Schemas['social.PostKind'];
export type SocialPostStatus = Schemas['social.PostStatus'];

export interface PaginatedResponse<T> {
	data: T[];
	page: number;
	per_page: number;
	total: number;
}

// Dashboard-only query helper. Not part of the API contract — the client maps
// these onto each endpoint's documented query parameters.
export type Bucket = 'day' | 'week' | 'month';
export type TrendMetric = 'unique_opens' | 'unique_clicks' | 'open_rate' | 'click_rate';

export interface MetricsQuery {
	from?: string;
	to?: string;
	bucket?: Bucket;
	metric?: TrendMetric;
	limit?: number;
}
