<script lang="ts">
	import { api } from '$lib/api/client';
	import type { DigestIssue, IssueEngagement, IssueStatus } from '$lib/api/types';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Table, THead, TBody, TR, TH, TD } from '$lib/components/ui/table';
	import { Badge } from '$lib/components/ui/badge';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Button } from '$lib/components/ui/button';
	import KpiCard from '$lib/components/kpi-card.svelte';
	import { dateRange, toQueryParams } from '$lib/stores/dateRange';
	import { formatCompact, formatPercent, formatDate } from '$lib/utils/format';
	import { toast } from 'svelte-sonner';

	type Filter = 'all' | 'draft' | 'sent';

	interface Row {
		id: number;
		slug: string;
		subject: string;
		status: IssueStatus;
		sent_at: string;
		delivered?: number;
		unique_opens?: number;
		unique_clicks?: number;
		open_rate?: number;
		click_rate?: number;
		bounced?: number;
	}

	let filter = $state<Filter>('all');
	let rows = $state<Row[] | null>(null);
	let engagement = $state<IssueEngagement[] | null>(null);
	let loading = $state(true);
	let sortKey = $state<string>('sent_at');
	let sortDir = $state<'asc' | 'desc'>('desc');

	async function load() {
		loading = true;
		try {
			const status: IssueStatus | undefined = filter === 'all' ? undefined : filter;
			const q = toQueryParams($dateRange);
			const [issuesPage, eng] = await Promise.all([
				api.digestIssues(status, 1, 200),
				api.issues({ ...q, limit: 100 }).catch(() => [] as IssueEngagement[])
			]);
			engagement = eng;
			const bySlug = new Map(eng.map((e) => [e.slug, e]));
			rows = issuesPage.data.map((i: DigestIssue) => {
				const m = bySlug.get(i.slug);
				return {
					id: i.id,
					slug: i.slug,
					subject: i.subject,
					status: i.status,
					sent_at: i.sent_at,
					delivered: m?.delivered,
					unique_opens: m?.unique_opens,
					unique_clicks: m?.unique_clicks,
					open_rate: m?.open_rate,
					click_rate: m?.click_rate,
					bounced: m?.bounced
				};
			});
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
		filter;
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
		if (!rows) return [];
		const out = [...rows];
		out.sort((a, b) => {
			const av = (a as unknown as Record<string, unknown>)[sortKey];
			const bv = (b as unknown as Record<string, unknown>)[sortKey];
			if (typeof av === 'string' && typeof bv === 'string') {
				return sortDir === 'asc' ? av.localeCompare(bv) : bv.localeCompare(av);
			}
			const an = av == null ? -Infinity : Number(av);
			const bn = bv == null ? -Infinity : Number(bv);
			return sortDir === 'asc' ? an - bn : bn - an;
		});
		return out;
	});

	const summary = $derived.by(() => {
		const sent = (engagement ?? []).filter((e) =>
			(rows ?? []).some((r) => r.slug === e.slug && r.status === 'sent')
		);
		if (!sent.length) return null;
		return {
			total: sent.length,
			avgOpenRate: sent.reduce((s, r) => s + r.open_rate, 0) / sent.length,
			avgClickRate: sent.reduce((s, r) => s + r.click_rate, 0) / sent.length,
			totalDelivered: sent.reduce((s, r) => s + r.delivered, 0)
		};
	});

	function sortIcon(key: string) {
		if (sortKey !== key) return '↕';
		return sortDir === 'asc' ? '↑' : '↓';
	}

	function statusVariant(status: IssueStatus): 'default' | 'secondary' | 'success' | 'destructive' {
		if (status === 'sent') return 'success';
		if (status === 'error') return 'destructive';
		return 'secondary';
	}
</script>

<svelte:head><title>Issues | GoDaily Analytics</title></svelte:head>

<div class="space-y-6">
	<div>
		<h1 class="text-xl font-semibold tracking-tight">Issues</h1>
		<p class="text-muted-foreground text-sm mt-1">Drafts and sent issues. Click a row to view or edit.</p>
	</div>

	<div class="flex items-center gap-2">
		<Button variant={filter === 'all' ? 'default' : 'outline'} size="sm" onclick={() => (filter = 'all')}>
			All
		</Button>
		<Button variant={filter === 'draft' ? 'default' : 'outline'} size="sm" onclick={() => (filter = 'draft')}>
			Drafts
		</Button>
		<Button variant={filter === 'sent' ? 'default' : 'outline'} size="sm" onclick={() => (filter = 'sent')}>
			Sent
		</Button>
	</div>

	<!-- Summary KPIs (sent only) -->
	<div class="grid grid-cols-2 gap-4 sm:grid-cols-4">
		<KpiCard
			label="Sent in view"
			value={summary ? formatCompact(summary.total) : '--'}
			loading={loading && !rows}
		/>
		<KpiCard
			label="Total delivered"
			value={summary ? formatCompact(summary.totalDelivered) : '--'}
			loading={loading && !rows}
		/>
		<KpiCard
			label="Avg open rate"
			value={summary ? formatPercent(summary.avgOpenRate) : '--'}
			loading={loading && !rows}
		/>
		<KpiCard
			label="Avg click rate"
			value={summary ? formatPercent(summary.avgClickRate) : '--'}
			loading={loading && !rows}
		/>
	</div>

	<Card>
		<CardHeader>
			<CardTitle>All issues</CardTitle>
			<CardDescription>Click a column header to sort, or a row to open</CardDescription>
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
							<TH>Status</TH>
							<TH>
								<button onclick={() => toggleSort('sent_at')} class="flex items-center gap-1 hover:text-foreground">
									Date <span class="text-muted-foreground">{sortIcon('sent_at')}</span>
								</button>
							</TH>
							<TH class="text-right">
								<button onclick={() => toggleSort('delivered')} class="flex items-center gap-1 hover:text-foreground ml-auto">
									Delivered <span class="text-muted-foreground">{sortIcon('delivered')}</span>
								</button>
							</TH>
							<TH class="text-right">
								<button onclick={() => toggleSort('open_rate')} class="flex items-center gap-1 hover:text-foreground ml-auto">
									Open % <span class="text-muted-foreground">{sortIcon('open_rate')}</span>
								</button>
							</TH>
							<TH class="text-right">
								<button onclick={() => toggleSort('click_rate')} class="flex items-center gap-1 hover:text-foreground ml-auto">
									Click % <span class="text-muted-foreground">{sortIcon('click_rate')}</span>
								</button>
							</TH>
						</TR>
					</THead>
					<TBody>
						{#each sorted as r (r.id)}
							<TR>
								<TD class="whitespace-nowrap">
									<a href={`/issues/${r.id}`} class="hover:text-primary font-medium">
										{r.slug}
									</a>
								</TD>
								<TD>
									<Badge variant={statusVariant(r.status)}>{r.status}</Badge>
								</TD>
								<TD class="text-muted-foreground whitespace-nowrap text-xs">{formatDate(r.sent_at)}</TD>
								<TD class="text-right tabular-nums">
									{#if r.delivered != null}{formatCompact(r.delivered)}{:else}<span class="text-muted-foreground">—</span>{/if}
								</TD>
								<TD class="text-right">
									{#if r.open_rate != null}
										<Badge variant="secondary">{formatPercent(r.open_rate)}</Badge>
									{:else}
										<span class="text-muted-foreground">—</span>
									{/if}
								</TD>
								<TD class="text-right">
									{#if r.click_rate != null}
										<Badge variant="success">{formatPercent(r.click_rate)}</Badge>
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
