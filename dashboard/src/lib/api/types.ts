export interface SummaryStats {
	from: string;
	to: string;
	issues_sent: number;
	delivered: number;
	unique_opens: number;
	total_opens: number;
	unique_clicks: number;
	total_clicks: number;
	bounced: number;
	complained: number;
	open_rate: number;
	click_rate: number;
	unique_subscribers_engaged: number;
}

export interface IssueEngagement {
	issue_id: number;
	slug: string;
	sent_at: string;
	delivered: number;
	unique_opens: number;
	total_opens: number;
	unique_clicks: number;
	total_clicks: number;
	bounced: number;
	complained: number;
	delayed?: number;
	failed?: number;
	suppressed?: number;
	open_rate: number;
	click_rate: number;
}

export interface ItemMetrics {
	item_id: number;
	title: string;
	url: string;
	tag: string;
	source: string;
	clicks: number;
}

export interface TagMetrics {
	tag: string;
	clicks: number;
}

export interface SourceMetrics {
	source: string;
	clicks: number;
}

export interface TrendPoint {
	bucket_start: string;
	value: number;
	delivered: number;
}

export interface TrendData {
	metric: string;
	bucket: string;
	points: TrendPoint[];
}

export interface SubscriberPoint {
	bucket_start: string;
	new: number;
	confirmed: number;
	unsubscribed: number;
	lost: number;
	net_change: number;
	active_at_end: number;
}

export interface SubscriberData {
	bucket: string;
	points: SubscriberPoint[];
}

export interface SocialPostMetric {
	id: number;
	issue_id?: number;
	kind: string;
	subject?: string;
	platform: string;
	text: string;
	post_url?: string;
	posted_at: string;
	likes: number;
	reposts: number;
	comments: number;
	impressions: number;
}

export interface Subscriber {
	id: number;
	email: string;
	confirmed_at?: string;
	unsubscribed_at?: string;
	bounced_at?: string;
	suppressed_at?: string;
	created_at: string;
}

export type IssueStatus = 'draft' | 'sent' | 'error';

export interface DigestAuthor {
	name?: string;
	username?: string;
	avatar_url?: string;
	profile_url?: string;
}

export interface DigestItem {
	id: number;
	source: string;
	tag: string;
	title: string;
	url: string;
	original_url?: string;
	image_url?: string;
	snippet: string;
	score: number;
	comments?: number;
	published?: string;
	in_digest?: boolean;
	author?: DigestAuthor;
}

export interface DigestIssue {
	id: number;
	slug: string;
	subject: string;
	summary?: string;
	status: IssueStatus;
	sent_at: string;
	items: DigestItem[];
}

export interface PaginatedResponse<T> {
	data: T[];
	page: number;
	per_page: number;
	total: number;
}

export type Bucket = 'day' | 'week' | 'month';
export type TrendMetric = 'unique_opens' | 'unique_clicks' | 'open_rate' | 'click_rate';

export interface MetricsQuery {
	from?: string;
	to?: string;
	bucket?: Bucket;
	metric?: TrendMetric;
	limit?: number;
}
