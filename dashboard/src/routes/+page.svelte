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

	const activeSubs = $derived(
		subscribers && subscribers.points.length
			? subscribers.points[subscribers.points.length - 1].active_at_end
			: null
	);

	const netChange = $derived(
		subscribers ? subscribers.points.reduce((acc, p) => acc + p.net_change, 0) : null
	);

	const newSubs = $derived(
		subscribers ? subscribers.points.reduce((acc, p) => acc + p.new, 0) : null
	);

	const unsubs = $derived(
		subscribers ? subscribers.points.reduce((acc, p) => acc + p.unsubscribed, 0) : null
	);

	const engagementRate = $derived(
		summary && activeSubs && activeSubs > 0
			? summary.unique_subscribers_engaged / activeSubs
			: null
	);

	const clickToOpen = $derived(
		summary && summary.unique_opens > 0 ? summary.unique_clicks / summary.unique_opens : null
	);

	const netDelta = $derived.by(() => {
		if (netChange == null) return undefined;
		const sign = netChange > 0 ? '+' : '';
		return {
			value: `${sign}${formatCompact(netChange)}`,
			direction: netChange > 0 ? 'up' : netChange < 0 ? 'down' : 'flat'
		} as const;
	});
</script>

<svelte:head><title>Dashboard | GoDaily Analytics</title></svelte:head>

<div class="space-y-6">
	<!-- Hero stats -->
	<div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
		<KpiCard
			label="Active subscribers"
			size="lg"
			value={activeSubs != null ? formatCompact(activeSubs) : '--'}
			sublabel={newSubs != null && unsubs != null
				? `+${formatCompact(newSubs)} new · -${formatCompact(unsubs)} unsubscribed`
				: undefined}
			delta={netDelta}
			loading={loading && !subscribers}
		/>
		<KpiCard
			label="Total opens"
			size="lg"
			value={summary ? formatCompact(summary.total_opens) : '--'}
			sublabel={summary ? `${formatCompact(summary.unique_opens)} unique` : undefined}
			{loading}
		/>
		<KpiCard
			label="Total clicks"
			size="lg"
			value={summary ? formatCompact(summary.total_clicks) : '--'}
			sublabel={summary ? `${formatCompact(summary.unique_clicks)} unique` : undefined}
			{loading}
		/>
		<KpiCard
			label="Engagement rate"
			size="lg"
			value={engagementRate != null ? formatPercent(engagementRate) : '--'}
			sublabel={summary && activeSubs
				? `${formatCompact(summary.unique_subscribers_engaged)} of ${formatCompact(activeSubs)} active`
				: undefined}
			loading={loading && (!summary || !subscribers)}
		/>
	</div>

	<!-- Detail KPIs -->
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
			{loading}
		/>
		<KpiCard
			label="Click rate"
			value={summary ? formatPercent(summary.click_rate) : '--'}
			{loading}
		/>
		<KpiCard
			label="Click-to-open"
			value={clickToOpen != null ? formatPercent(clickToOpen) : '--'}
			sublabel="of opens that clicked"
			{loading}
		/>
		<KpiCard
			label="Bounce rate"
			value={bounceRate != null ? formatPercent(bounceRate) : '--'}
			sublabel={summary
				? `${formatCompact(summary.bounced)} bounced · ${formatCompact(summary.complained)} complaints`
				: undefined}
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
