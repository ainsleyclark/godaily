<script lang="ts">
	import { api } from '$lib/api/client';
	import type { IssueEngagement } from '$lib/api/types';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Table, THead, TBody, TR, TH, TD } from '$lib/components/ui/table';
	import { Badge } from '$lib/components/ui/badge';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import KpiCard from '$lib/components/kpi-card.svelte';
	import { dateRange, toQueryParams } from '$lib/stores/dateRange';
	import { formatCompact, formatPercent, formatDate } from '$lib/utils/format';
	import { toast } from 'svelte-sonner';

	let issues = $state<IssueEngagement[] | null>(null);
	let loading = $state(true);
	let sortKey = $state<string>('sent_at');
	let sortDir = $state<'asc' | 'desc'>('desc');

	async function load() {
		const q = toQueryParams($dateRange);
		loading = true;
		try {
			issues = await api.issues({ ...q, limit: 100 });
		} catch (e) {
			if ((e as { status?: number }).status !== 401) {
				toast.error('Failed to load issues');
			}
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		$dateRange;
		void load();
	});

	function toggleSort(key: string) {
		if (sortKey === key) {
			sortDir = sortDir === 'asc' ? 'desc' : 'asc';
		} else {
			sortKey = key;
			sortDir = 'desc';
		}
	}

	const sorted = $derived.by(() => {
		if (!issues) return [];
		const rows = [...issues];
		rows.sort((a, b) => {
			const av = (a as unknown as Record<string, unknown>)[sortKey];
			const bv = (b as unknown as Record<string, unknown>)[sortKey];
			if (typeof av === 'string' && typeof bv === 'string') {
				return sortDir === 'asc' ? av.localeCompare(bv) : bv.localeCompare(av);
			}
			const an = Number(av ?? 0);
			const bn = Number(bv ?? 0);
			return sortDir === 'asc' ? an - bn : bn - an;
		});
		return rows;
	});

	const summary = $derived.by(() => {
		if (!issues || !issues.length) return null;
		return {
			total: issues.length,
			avgOpenRate: issues.reduce((s, r) => s + r.open_rate, 0) / issues.length,
			avgClickRate: issues.reduce((s, r) => s + r.click_rate, 0) / issues.length,
			totalDelivered: issues.reduce((s, r) => s + r.delivered, 0)
		};
	});

	function sortIcon(key: string) {
		if (sortKey !== key) return '↕';
		return sortDir === 'asc' ? '↑' : '↓';
	}
</script>

<svelte:head><title>Issues | GoDaily Analytics</title></svelte:head>

<div class="space-y-6">
	<div>
		<h1 class="text-xl font-semibold tracking-tight">Issues</h1>
		<p class="text-muted-foreground text-sm mt-1">Full engagement breakdown for every sent issue</p>
	</div>

	<!-- Summary KPIs -->
	<div class="grid grid-cols-2 gap-4 sm:grid-cols-4">
		<KpiCard
			label="Issues sent"
			value={summary ? formatCompact(summary.total) : '--'}
			loading={loading && !issues}
		/>
		<KpiCard
			label="Total delivered"
			value={summary ? formatCompact(summary.totalDelivered) : '--'}
			loading={loading && !issues}
		/>
		<KpiCard
			label="Avg open rate"
			value={summary ? formatPercent(summary.avgOpenRate) : '--'}
			loading={loading && !issues}
		/>
		<KpiCard
			label="Avg click rate"
			value={summary ? formatPercent(summary.avgClickRate) : '--'}
			loading={loading && !issues}
		/>
	</div>

	<!-- Issues table -->
	<Card>
		<CardHeader>
			<CardTitle>All issues</CardTitle>
			<CardDescription>Click a column header to sort</CardDescription>
		</CardHeader>
		<CardContent class="p-0">
			{#if loading && !sorted.length}
				<div class="space-y-2 p-4">
					{#each Array(8) as _, i (i)}
						<Skeleton class="h-10 w-full" />
					{/each}
				</div>
			{:else if !sorted.length}
				<div class="text-muted-foreground p-8 text-center text-sm">No issues found</div>
			{:else}
				<Table>
					<THead>
						<TR>
							<TH>
								<button onclick={() => toggleSort('slug')} class="flex items-center gap-1 hover:text-foreground">
									Issue <span class="text-muted-foreground">{sortIcon('slug')}</span>
								</button>
							</TH>
							<TH>
								<button onclick={() => toggleSort('sent_at')} class="flex items-center gap-1 hover:text-foreground ml-auto">
									Date <span class="text-muted-foreground">{sortIcon('sent_at')}</span>
								</button>
							</TH>
							<TH class="text-right">
								<button onclick={() => toggleSort('delivered')} class="flex items-center gap-1 hover:text-foreground ml-auto">
									Delivered <span class="text-muted-foreground">{sortIcon('delivered')}</span>
								</button>
							</TH>
							<TH class="text-right">
								<button onclick={() => toggleSort('unique_opens')} class="flex items-center gap-1 hover:text-foreground ml-auto">
									Opens <span class="text-muted-foreground">{sortIcon('unique_opens')}</span>
								</button>
							</TH>
							<TH class="text-right">
								<button onclick={() => toggleSort('open_rate')} class="flex items-center gap-1 hover:text-foreground ml-auto">
									Open % <span class="text-muted-foreground">{sortIcon('open_rate')}</span>
								</button>
							</TH>
							<TH class="text-right">
								<button onclick={() => toggleSort('unique_clicks')} class="flex items-center gap-1 hover:text-foreground ml-auto">
									Clicks <span class="text-muted-foreground">{sortIcon('unique_clicks')}</span>
								</button>
							</TH>
							<TH class="text-right">
								<button onclick={() => toggleSort('click_rate')} class="flex items-center gap-1 hover:text-foreground ml-auto">
									Click % <span class="text-muted-foreground">{sortIcon('click_rate')}</span>
								</button>
							</TH>
							<TH class="text-right">
								<button onclick={() => toggleSort('bounced')} class="flex items-center gap-1 hover:text-foreground ml-auto">
									Bounced <span class="text-muted-foreground">{sortIcon('bounced')}</span>
								</button>
							</TH>
						</TR>
					</THead>
					<TBody>
						{#each sorted as r (r.issue_id)}
							<TR>
								<TD>
									<a
										href={`https://godaily.dev/issues/${r.slug}`}
										target="_blank"
										rel="noopener"
										class="hover:text-primary font-medium"
									>
										{r.slug}
									</a>
								</TD>
								<TD class="text-muted-foreground text-xs">{formatDate(r.sent_at)}</TD>
								<TD class="text-right tabular-nums">{formatCompact(r.delivered)}</TD>
								<TD class="text-right tabular-nums">{formatCompact(r.unique_opens)}</TD>
								<TD class="text-right">
									<Badge variant="secondary">{formatPercent(r.open_rate)}</Badge>
								</TD>
								<TD class="text-right tabular-nums">{formatCompact(r.unique_clicks)}</TD>
								<TD class="text-right">
									<Badge variant="success">{formatPercent(r.click_rate)}</Badge>
								</TD>
								<TD class="text-right tabular-nums text-sm">
									{#if r.bounced > 0}
										<span class="text-destructive">{formatCompact(r.bounced)}</span>
									{:else}
										<span class="text-muted-foreground">—</span>
									{/if}
								</TD>
							</TR>
						{/each}
					</TBody>
				</Table>
			{/if}
		</CardContent>
	</Card>
</div>
