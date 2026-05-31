<script lang="ts">
	import { api } from '$lib/api/client';
	import type { Subscriber, SubscriberData, PaginatedResponse } from '$lib/api/types';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Table, THead, TBody, TR, TH, TD } from '$lib/components/ui/table';
	import { Badge } from '$lib/components/ui/badge';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import KpiCard from '$lib/components/kpi-card.svelte';
	import SubscriberGrowth from '$lib/components/charts/subscriber-growth.svelte';
	import { dateRange, toQueryParams } from '$lib/stores/dateRange';
	import { formatCompact, formatDate } from '$lib/utils/format';
	import { toast } from 'svelte-sonner';

	let growth = $state<SubscriberData | null>(null);
	let listData = $state<PaginatedResponse<Subscriber> | null>(null);
	let loadingGrowth = $state(true);
	let loadingList = $state(true);
	let page = $state(1);
	let search = $state('');
	let searchInput = $state('');
	let updatingId = $state<number | null>(null);
	const perPage = 50;

	async function loadGrowth() {
		const q = toQueryParams($dateRange);
		loadingGrowth = true;
		try {
			growth = await api.subscribers(q);
		} catch (e) {
			if ((e as { status?: number }).status !== 401) toast.error('Failed to load subscriber data');
		} finally {
			loadingGrowth = false;
		}
	}

	async function loadList(p: number, s: string) {
		loadingList = true;
		try {
			listData = await api.subscriberList(p, perPage, s);
		} catch (e) {
			if ((e as { status?: number }).status !== 401) toast.error('Failed to load subscriber list');
		} finally {
			loadingList = false;
		}
	}

	$effect(() => {
		$dateRange;
		void loadGrowth();
	});

	$effect(() => {
		void loadList(page, search);
	});

	function submitSearch(e: SubmitEvent) {
		e.preventDefault();
		search = searchInput.trim();
		page = 1;
	}

	function clearSearch() {
		searchInput = '';
		search = '';
		page = 1;
	}

	async function setStatus(sub: Subscriber, status: string) {
		updatingId = sub.id;
		try {
			const updated = await api.updateSubscriberStatus(sub.id, status);
			if (listData) {
				listData = {
					...listData,
					data: listData.data.map((s) => (s.id === updated.id ? updated : s))
				};
			}
			toast.success(`${sub.email} set to ${status}`);
		} catch {
			toast.error('Failed to update subscriber status');
		} finally {
			updatingId = null;
		}
	}

	const points = $derived(growth?.points ?? []);

	const activeSubs = $derived(
		points.length ? points[points.length - 1].active_at_end : null
	);
	const totalNew = $derived(points.reduce((s, p) => s + p.new, 0));
	const totalUnsub = $derived(points.reduce((s, p) => s + p.unsubscribed, 0));
	const netChange = $derived(points.reduce((s, p) => s + p.net_change, 0));

	const netDelta = $derived.by(() => {
		if (!netChange && netChange !== 0) return undefined;
		const sign = netChange > 0 ? '+' : '';
		return {
			value: `${sign}${formatCompact(netChange)}`,
			direction: netChange > 0 ? 'up' : netChange < 0 ? 'down' : 'flat'
		} as const;
	});

	const totalPages = $derived(listData ? Math.ceil(listData.total / perPage) : 1);

	function statusOf(s: Subscriber): { label: string; variant: 'default' | 'secondary' | 'outline' | 'destructive' | 'success' } {
		if (s.bounced_at) return { label: 'Bounced', variant: 'destructive' };
		if (s.suppressed_at) return { label: 'Suppressed', variant: 'destructive' };
		if (s.unsubscribed_at) return { label: 'Unsubscribed', variant: 'secondary' };
		if (!s.confirmed_at) return { label: 'Pending', variant: 'outline' };
		return { label: 'Active', variant: 'success' };
	}

	function availableStatuses(s: Subscriber): { value: string; label: string }[] {
		const current = statusOf(s).label.toLowerCase();
		const all = [
			{ value: 'active', label: 'Active' },
			{ value: 'unsubscribed', label: 'Unsubscribed' },
			{ value: 'suppressed', label: 'Suppressed' }
		];
		return all.filter((st) => st.value !== current && current !== 'bounced');
	}
</script>

<svelte:head><title>Subscribers | GoDaily Analytics</title></svelte:head>

<div class="space-y-6">
	<div>
		<h1 class="text-xl font-semibold tracking-tight">Subscribers</h1>
		<p class="text-muted-foreground text-sm mt-1">Growth, churn, and subscriber list</p>
	</div>

	<!-- KPIs -->
	<div class="grid grid-cols-2 gap-4 sm:grid-cols-4">
		<KpiCard
			label="Active"
			value={activeSubs != null ? formatCompact(activeSubs) : '--'}
			delta={netDelta}
			loading={loadingGrowth && !growth}
		/>
		<KpiCard
			label="New"
			value={formatCompact(totalNew)}
			loading={loadingGrowth && !growth}
		/>
		<KpiCard
			label="Unsubscribed"
			value={formatCompact(totalUnsub)}
			loading={loadingGrowth && !growth}
		/>
		<KpiCard
			label="Total list"
			value={listData ? formatCompact(listData.total) : '--'}
			loading={loadingList && !listData}
		/>
	</div>

	<!-- Growth chart -->
	<SubscriberGrowth data={growth} loading={loadingGrowth && !growth} />

	<!-- Subscriber list -->
	<Card>
		<CardHeader>
			<div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
				<div>
					<CardTitle>All subscribers</CardTitle>
					<CardDescription>
						{#if listData}
							{formatCompact(listData.total)} total{search ? ` matching "${search}"` : ''} · page {page} of {totalPages}
						{:else}
							Loading…
						{/if}
					</CardDescription>
				</div>
				<div class="flex flex-wrap items-center gap-2">
					<!-- Search -->
					<form onsubmit={submitSearch} class="flex items-center gap-1.5">
						<input
							type="search"
							bind:value={searchInput}
							placeholder="Search email…"
							class="border-border bg-background h-8 rounded-md border px-3 text-sm focus:outline-none focus:ring-1 focus:ring-ring w-44"
						/>
						<button
							type="submit"
							class="border-border bg-background h-8 rounded-md border px-3 text-xs hover:bg-accent"
						>
							Search
						</button>
						{#if search}
							<button
								type="button"
								onclick={clearSearch}
								class="border-border bg-background h-8 rounded-md border px-3 text-xs hover:bg-accent text-muted-foreground"
							>
								Clear
							</button>
						{/if}
					</form>
					<!-- Pagination -->
					{#if totalPages > 1}
						<div class="flex items-center gap-2">
							<button
								onclick={() => (page = Math.max(1, page - 1))}
								disabled={page === 1 || loadingList}
								class="rounded-md border px-3 py-1.5 text-xs disabled:opacity-40 hover:bg-accent"
							>
								← Prev
							</button>
							<button
								onclick={() => (page = Math.min(totalPages, page + 1))}
								disabled={page === totalPages || loadingList}
								class="rounded-md border px-3 py-1.5 text-xs disabled:opacity-40 hover:bg-accent"
							>
								Next →
							</button>
						</div>
					{/if}
				</div>
			</div>
		</CardHeader>
		<CardContent class="p-0">
			{#if loadingList}
				<div class="space-y-2 p-4">
					{#each Array(8) as _, i (i)}
						<Skeleton class="h-10 w-full" />
					{/each}
				</div>
			{:else if !listData?.data.length}
				<div class="text-muted-foreground p-8 text-center text-sm">
					{search ? `No subscribers matching "${search}"` : 'No subscribers found'}
				</div>
			{:else}
				<Table>
					<THead>
						<TR>
							<TH>Email</TH>
							<TH>Status</TH>
							<TH>Subscribed</TH>
							<TH>Confirmed</TH>
							<TH>Actions</TH>
						</TR>
					</THead>
					<TBody>
						{#each listData.data as s (s.id)}
							{@const status = statusOf(s)}
							{@const actions = availableStatuses(s)}
							<TR>
								<TD class="font-mono text-sm">{s.email}</TD>
								<TD>
									<Badge variant={status.variant}>{status.label}</Badge>
								</TD>
								<TD class="text-muted-foreground text-xs">{formatDate(s.created_at)}</TD>
								<TD class="text-muted-foreground text-xs">
									{s.confirmed_at ? formatDate(s.confirmed_at) : '—'}
								</TD>
								<TD>
									{#if updatingId === s.id}
										<span class="text-muted-foreground text-xs">Saving…</span>
									{:else if actions.length > 0}
										<div class="flex flex-wrap gap-1">
											{#each actions as action (action.value)}
												<button
													onclick={() => setStatus(s, action.value)}
													class="rounded border px-2 py-0.5 text-xs hover:bg-accent transition-colors"
												>
													{action.label}
												</button>
											{/each}
										</div>
									{:else}
										<span class="text-muted-foreground text-xs">—</span>
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
