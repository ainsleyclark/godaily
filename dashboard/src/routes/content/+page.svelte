<script lang="ts">
	import { api } from '$lib/api/client';
	import type { ItemMetrics, TagMetrics, SourceMetrics } from '$lib/api/types';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Table, THead, TBody, TR, TH, TD } from '$lib/components/ui/table';
	import { Badge } from '$lib/components/ui/badge';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import KpiCard from '$lib/components/kpi-card.svelte';
	import { dateRange, toQueryParams } from '$lib/stores/dateRange';
	import { formatCompact } from '$lib/utils/format';
	import { prettify } from '$lib/utils/labels';
	import { toast } from 'svelte-sonner';

	let items = $state<ItemMetrics[] | null>(null);
	let tags = $state<TagMetrics[] | null>(null);
	let sources = $state<SourceMetrics[] | null>(null);
	let loading = $state(true);

	let search = $state('');
	let tagFilter = $state('all');
	let sourceFilter = $state('all');

	async function load() {
		const q = toQueryParams($dateRange);
		loading = true;
		try {
			[items, tags, sources] = await Promise.all([
				api.items({ ...q, limit: 100 }),
				api.tags(q),
				api.sources(q)
			]);
		} catch (e) {
			if ((e as { status?: number }).status !== 401) toast.error('Failed to load content data');
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		$dateRange;
		void load();
	});

	const allTags = $derived([...new Set((items ?? []).map((i) => i.tag).filter(Boolean))].sort());
	const allSources = $derived([...new Set((items ?? []).map((i) => i.source).filter(Boolean))].sort());

	const filtered = $derived.by(() => {
		let rows = items ?? [];
		if (tagFilter !== 'all') rows = rows.filter((r) => r.tag === tagFilter);
		if (sourceFilter !== 'all') rows = rows.filter((r) => r.source === sourceFilter);
		if (search.trim()) {
			const q = search.trim().toLowerCase();
			rows = rows.filter(
				(r) => r.title.toLowerCase().includes(q) || r.source.toLowerCase().includes(q)
			);
		}
		return rows;
	});

	const totalClicks = $derived((items ?? []).reduce((s, r) => s + r.clicks, 0));
	const maxClicks = $derived(Math.max(1, ...(items ?? []).map((r) => r.clicks)));
</script>

<div class="space-y-6">
	<div>
		<h1 class="text-xl font-semibold tracking-tight">Content</h1>
		<p class="text-muted-foreground text-sm mt-1">All tracked links with click performance</p>
	</div>

	<!-- KPIs -->
	<div class="grid grid-cols-2 gap-4 sm:grid-cols-4">
		<KpiCard
			label="Total items"
			value={items ? formatCompact(items.length) : '--'}
			loading={loading && !items}
		/>
		<KpiCard
			label="Total clicks"
			value={formatCompact(totalClicks)}
			loading={loading && !items}
		/>
		<KpiCard
			label="Tags"
			value={tags ? formatCompact(tags.length) : '--'}
			loading={loading && !tags}
		/>
		<KpiCard
			label="Sources"
			value={sources ? formatCompact(sources.length) : '--'}
			loading={loading && !sources}
		/>
	</div>

	<!-- Filters -->
	<Card>
		<CardHeader>
			<div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
				<div>
					<CardTitle>All items</CardTitle>
					<CardDescription>
						{filtered.length} of {items?.length ?? 0} items
					</CardDescription>
				</div>
				<div class="flex flex-wrap items-center gap-2">
					<!-- Search -->
					<input
						type="search"
						bind:value={search}
						placeholder="Search…"
						class="border-border bg-background h-8 rounded-md border px-3 text-sm focus:outline-none focus:ring-1 focus:ring-ring w-40"
					/>
					<!-- Tag filter -->
					{#if allTags.length > 1}
						<select
							bind:value={tagFilter}
							class="border-border bg-background h-8 rounded-md border px-2 text-sm focus:outline-none focus:ring-1 focus:ring-ring"
						>
							<option value="all">All tags</option>
							{#each allTags as t (t)}
								<option value={t}>{prettify(t)}</option>
							{/each}
						</select>
					{/if}
					<!-- Source filter -->
					{#if allSources.length > 1}
						<select
							bind:value={sourceFilter}
							class="border-border bg-background h-8 rounded-md border px-2 text-sm focus:outline-none focus:ring-1 focus:ring-ring"
						>
							<option value="all">All sources</option>
							{#each allSources as s (s)}
								<option value={s}>{prettify(s)}</option>
							{/each}
						</select>
					{/if}
				</div>
			</div>
		</CardHeader>
		<CardContent class="p-0">
			{#if loading && !filtered.length}
				<div class="space-y-2 p-4">
					{#each Array(8) as _, i (i)}
						<Skeleton class="h-10 w-full" />
					{/each}
				</div>
			{:else if !filtered.length}
				<div class="text-muted-foreground p-8 text-center text-sm">No items match your filters</div>
			{:else}
				<Table>
					<THead>
						<TR>
							<TH>Title</TH>
							<TH>Tag</TH>
							<TH>Source</TH>
							<TH class="text-right">Clicks</TH>
							<TH class="text-right w-32">Share</TH>
						</TR>
					</THead>
					<TBody>
						{#each filtered as r (r.item_id)}
							<TR>
								<TD>
									<a
										href={r.url}
										target="_blank"
										rel="noopener"
										class="hover:text-primary block max-w-[340px] truncate font-medium text-sm"
										title={r.title}
									>
										{r.title}
									</a>
								</TD>
								<TD>
									{#if r.tag}
										<Badge variant="outline">{prettify(r.tag)}</Badge>
									{:else}
										<span class="text-muted-foreground text-xs">—</span>
									{/if}
								</TD>
								<TD class="text-muted-foreground text-xs">{prettify(r.source)}</TD>
								<TD class="text-right tabular-nums font-medium">{formatCompact(r.clicks)}</TD>
								<TD class="text-right">
									<div class="flex items-center justify-end gap-2">
										<div class="bg-muted h-1.5 w-24 overflow-hidden rounded-full">
											<div
												class="bg-primary h-full rounded-full"
												style="width: {Math.round((r.clicks / maxClicks) * 100)}%"
											></div>
										</div>
										<span class="text-muted-foreground w-8 text-right text-xs tabular-nums">
											{Math.round((r.clicks / totalClicks) * 100)}%
										</span>
									</div>
								</TD>
							</TR>
						{/each}
					</TBody>
				</Table>
			{/if}
		</CardContent>
	</Card>
</div>
