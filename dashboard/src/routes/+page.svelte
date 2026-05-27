<script lang="ts">
	import { api } from '$lib/api/client';
	import type {
		IssueEngagement,
		ItemMetrics,
		SourceMetrics,
		SubscriberData,
		SummaryStats,
		TagMetrics,
		TrendData,
		TrendMetric
	} from '$lib/api/types';
	import KpiCard from '$lib/components/kpi-card.svelte';
	import EngagementTrend from '$lib/components/charts/engagement-trend.svelte';
	import SubscriberGrowth from '$lib/components/charts/subscriber-growth.svelte';
	import TopIssues from '$lib/components/tables/top-issues.svelte';
	import TopItems from '$lib/components/tables/top-items.svelte';
	import TopTags from '$lib/components/tables/top-tags.svelte';
	import TopSources from '$lib/components/tables/top-sources.svelte';
	import { dateRange, toQueryParams } from '$lib/stores/dateRange';
	import { formatCompact, formatPercent } from '$lib/utils/format';
	import { toast } from 'svelte-sonner';

	let summary = $state<SummaryStats | null>(null);
	let issues = $state<IssueEngagement[] | null>(null);
	let items = $state<ItemMetrics[] | null>(null);
	let tags = $state<TagMetrics[] | null>(null);
	let sources = $state<SourceMetrics[] | null>(null);
	let trend = $state<TrendData | null>(null);
	let subscribers = $state<SubscriberData | null>(null);
	let loading = $state(true);
	let trendMetric = $state<TrendMetric>('unique_clicks');

	async function loadAll() {
		const q = toQueryParams($dateRange);
		loading = true;
		try {
			const [s, i, it, tg, sr, tr, sb] = await Promise.all([
				api.summary(q),
				api.issues(q),
				api.items({ ...q, limit: 10 }),
				api.tags(q),
				api.sources(q),
				api.trend({ ...q, metric: trendMetric }),
				api.subscribers(q)
			]);
			summary = s;
			issues = i;
			items = it;
			tags = tg;
			sources = sr;
			trend = tr;
			subscribers = sb;
		} catch (e) {
			if ((e as { status?: number }).status !== 401) {
				toast.error('Failed to load metrics');
				// eslint-disable-next-line no-console
				console.error(e);
			}
		} finally {
			loading = false;
		}
	}

	async function reloadTrend(m: TrendMetric) {
		trendMetric = m;
		const q = { ...toQueryParams($dateRange), metric: m };
		try {
			trend = await api.trend(q);
		} catch (e) {
			if ((e as { status?: number }).status !== 401) toast.error('Failed to load trend');
		}
	}

	$effect(() => {
		// re-run when dateRange changes
		$dateRange;
		void loadAll();
	});

	const bounceRate = $derived(
		summary && summary.delivered > 0 ? summary.bounced / summary.delivered : null
	);
</script>

<div class="space-y-6">
	<!-- KPI Row -->
	<div class="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-6">
		<KpiCard
			label="Issues sent"
			value={summary ? formatCompact(summary.issues_sent) : '--'}
			{loading}
		/>
		<KpiCard
			label="Delivered"
			value={summary ? formatCompact(summary.delivered) : '--'}
			{loading}
		/>
		<KpiCard
			label="Open rate"
			value={summary ? formatPercent(summary.open_rate) : '--'}
			sublabel={summary ? `${formatCompact(summary.unique_opens)} unique` : undefined}
			{loading}
		/>
		<KpiCard
			label="Click rate"
			value={summary ? formatPercent(summary.click_rate) : '--'}
			sublabel={summary ? `${formatCompact(summary.unique_clicks)} unique` : undefined}
			{loading}
		/>
		<KpiCard
			label="Engaged subs"
			value={summary ? formatCompact(summary.unique_subscribers_engaged) : '--'}
			{loading}
		/>
		<KpiCard
			label="Bounce rate"
			value={bounceRate != null ? formatPercent(bounceRate) : '--'}
			sublabel={summary ? `${formatCompact(summary.bounced)} bounced` : undefined}
			{loading}
		/>
	</div>

	<!-- Charts Row -->
	<div class="grid grid-cols-1 gap-4 lg:grid-cols-2">
		<EngagementTrend
			data={trend}
			loading={loading && !trend}
			metric={trendMetric}
			onMetricChange={reloadTrend}
		/>
		<SubscriberGrowth data={subscribers} loading={loading && !subscribers} />
	</div>

	<!-- Tables grid -->
	<div class="grid grid-cols-1 gap-4 lg:grid-cols-2">
		<TopIssues data={issues} loading={loading && !issues} />
		<TopItems data={items} loading={loading && !items} />
		<TopTags data={tags} loading={loading && !tags} />
		<TopSources data={sources} loading={loading && !sources} />
	</div>
</div>
