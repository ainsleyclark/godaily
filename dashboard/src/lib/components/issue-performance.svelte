<script lang="ts">
	import { api, ApiError } from '$lib/api/client';
	import type { IssueDetail, TrendData, TrendMetric } from '$lib/api/types';
	import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
	import KpiCard from '$lib/components/kpi-card.svelte';
	import EngagementTrend from '$lib/components/charts/engagement-trend.svelte';
	import TopLinks from '$lib/components/tables/top-links.svelte';
	import { formatCompact, formatPercent } from '$lib/utils/format';
	import { toast } from 'svelte-sonner';

	interface Props {
		slug: string;
	}
	let { slug }: Props = $props();

	let detail = $state<IssueDetail | null>(null);
	let trend = $state<TrendData | null>(null);
	let loading = $state(true);
	let trendLoading = $state(false);
	let metric = $state<TrendMetric>('unique_clicks');

	const stats = $derived(detail?.stats ?? null);

	// Click-to-open and bounce rates are derived; the API exposes the raw counts.
	const clickToOpen = $derived(
		stats && stats.unique_opens > 0 ? stats.unique_clicks / stats.unique_opens : null
	);
	const bounceRate = $derived(stats && stats.delivered > 0 ? stats.bounced / stats.delivered : 0);

	// Funnel stages, widths relative to delivered (the top of the funnel).
	const funnel = $derived(
		stats
			? [
					{ label: 'Delivered', value: stats.delivered },
					{ label: 'Opened', value: stats.unique_opens },
					{ label: 'Clicked', value: stats.unique_clicks }
				]
			: []
	);
	const funnelMax = $derived(funnel.length ? Math.max(funnel[0].value, 1) : 1);

	async function loadDetail() {
		loading = true;
		try {
			detail = await api.issueDetail(slug);
		} catch (e) {
			if ((e as ApiError).status !== 401) {
				toast.error((e as Error).message || 'Failed to load performance');
			}
		} finally {
			loading = false;
		}
	}

	async function loadTrend() {
		trendLoading = true;
		try {
			trend = await api.issueTrend(slug, { metric });
		} catch (e) {
			if ((e as ApiError).status !== 401) {
				toast.error((e as Error).message || 'Failed to load trend');
			}
		} finally {
			trendLoading = false;
		}
	}

	function onMetricChange(m: TrendMetric) {
		metric = m;
		void loadTrend();
	}

	$effect(() => {
		if (slug) {
			void loadDetail();
			void loadTrend();
		}
	});
</script>

<div class="space-y-6">
	<div class="grid grid-cols-2 gap-4 lg:grid-cols-5">
		<KpiCard label="Delivered" value={formatCompact(stats?.delivered)} {loading} />
		<KpiCard label="Open rate" value={formatPercent(stats?.open_rate)} {loading} />
		<KpiCard label="Click rate" value={formatPercent(stats?.click_rate)} {loading} />
		<KpiCard label="Click-to-open" value={formatPercent(clickToOpen)} {loading} />
		<KpiCard label="Bounce rate" value={formatPercent(bounceRate)} {loading} />
	</div>

	<EngagementTrend data={trend} loading={trendLoading} {metric} {onMetricChange} />

	<div class="grid gap-6 lg:grid-cols-2">
		<Card>
			<CardHeader>
				<CardTitle>Engagement funnel</CardTitle>
			</CardHeader>
			<CardContent class="space-y-4">
				{#each funnel as stage (stage.label)}
					<div class="space-y-1">
						<div class="flex items-baseline justify-between text-sm">
							<span class="font-medium">{stage.label}</span>
							<span class="text-muted-foreground tabular-nums">
								{formatCompact(stage.value)}
								<span class="text-xs">({formatPercent(stage.value / funnelMax)})</span>
							</span>
						</div>
						<div class="bg-secondary/40 h-2 w-full overflow-hidden rounded-full">
							<div
								class="bg-primary h-full rounded-full"
								style={`width: ${Math.min(100, (stage.value / funnelMax) * 100)}%`}
							></div>
						</div>
					</div>
				{/each}
				{#if !funnel.length && !loading}
					<div class="text-muted-foreground py-8 text-center text-sm">No data yet</div>
				{/if}
			</CardContent>
		</Card>

		<TopLinks data={detail?.links ?? null} {loading} />
	</div>
</div>
