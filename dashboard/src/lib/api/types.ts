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

export type Bucket = 'day' | 'week' | 'month';
export type TrendMetric = 'unique_opens' | 'unique_clicks' | 'open_rate' | 'click_rate';

export interface MetricsQuery {
	from?: string;
	to?: string;
	bucket?: Bucket;
	metric?: TrendMetric;
	limit?: number;
}
