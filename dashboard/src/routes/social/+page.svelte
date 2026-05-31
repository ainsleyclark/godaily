<script lang="ts">
	import { api } from '$lib/api/client';
	import type { SocialPostMetric } from '$lib/api/types';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Table, THead, TBody, TR, TH, TD } from '$lib/components/ui/table';
	import { Badge } from '$lib/components/ui/badge';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import KpiCard from '$lib/components/kpi-card.svelte';
	import { dateRange, toQueryParams } from '$lib/stores/dateRange';
	import { formatCompact, formatDate } from '$lib/utils/format';
	import { toast } from 'svelte-sonner';

	let posts = $state<SocialPostMetric[] | null>(null);
	let loading = $state(true);
	let platformFilter = $state<string>('all');

	async function load() {
		const q = toQueryParams($dateRange);
		loading = true;
		try {
			posts = await api.social(q);
		} catch (e) {
			if ((e as { status?: number }).status !== 401) {
				toast.error('Failed to load social metrics');
			}
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		$dateRange;
		void load();
	});

	const platforms = $derived([...new Set((posts ?? []).map((p) => p.platform))].sort());

	const filtered = $derived(
		platformFilter === 'all' ? (posts ?? []) : (posts ?? []).filter((p) => p.platform === platformFilter)
	);

	const totals = $derived.by(() => {
		const rows = posts ?? [];
		return {
			posts: rows.length,
			likes: rows.reduce((s, r) => s + r.likes, 0),
			reposts: rows.reduce((s, r) => s + r.reposts, 0),
			comments: rows.reduce((s, r) => s + r.comments, 0),
			impressions: rows.reduce((s, r) => s + r.impressions, 0)
		};
	});

	const byPlatform = $derived.by(() => {
		const map: Record<string, { likes: number; reposts: number; comments: number; impressions: number; posts: number }> = {};
		for (const p of posts ?? []) {
			if (!map[p.platform]) map[p.platform] = { likes: 0, reposts: 0, comments: 0, impressions: 0, posts: 0 };
			map[p.platform].likes += p.likes;
			map[p.platform].reposts += p.reposts;
			map[p.platform].comments += p.comments;
			map[p.platform].impressions += p.impressions;
			map[p.platform].posts += 1;
		}
		return Object.entries(map)
			.map(([platform, stats]) => ({ platform, ...stats }))
			.sort((a, b) => b.impressions - a.impressions);
	});

	const platformColors: Record<string, string> = {
		bluesky: 'bg-sky-500',
		mastodon: 'bg-violet-500',
		twitter: 'bg-blue-400',
		linkedin: 'bg-blue-700'
	};

	function platformColor(p: string) {
		return platformColors[p.toLowerCase()] ?? 'bg-muted-foreground';
	}

	function kindLabel(kind: string) {
		const labels: Record<string, string> = {
			featured: 'Featured',
			recap: 'Recap',
			spotlight: 'Spotlight',
			cta: 'CTA',
			community: 'Community',
			new_source: 'New Source'
		};
		return labels[kind] ?? kind;
	}

	function kindVariant(kind: string): 'default' | 'secondary' | 'outline' | 'success' {
		if (kind === 'featured') return 'default';
		if (kind === 'recap') return 'success';
		return 'secondary';
	}
</script>

<div class="space-y-6">
	<div>
		<h1 class="text-xl font-semibold tracking-tight">Social</h1>
		<p class="text-muted-foreground text-sm mt-1">Engagement metrics for social posts across all platforms</p>
	</div>

	<!-- Summary KPIs -->
	<div class="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-5">
		<KpiCard
			label="Total posts"
			value={formatCompact(totals.posts)}
			loading={loading && !posts}
		/>
		<KpiCard
			label="Impressions"
			value={formatCompact(totals.impressions)}
			loading={loading && !posts}
		/>
		<KpiCard
			label="Likes"
			value={formatCompact(totals.likes)}
			loading={loading && !posts}
		/>
		<KpiCard
			label="Reposts"
			value={formatCompact(totals.reposts)}
			loading={loading && !posts}
		/>
		<KpiCard
			label="Comments"
			value={formatCompact(totals.comments)}
			loading={loading && !posts}
		/>
	</div>

	<!-- By platform breakdown -->
	{#if byPlatform.length > 0}
		<div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
			{#each byPlatform as p (p.platform)}
				<Card>
					<CardHeader class="pb-2">
						<div class="flex items-center gap-2">
							<span class="inline-block h-2.5 w-2.5 rounded-full {platformColor(p.platform)}"></span>
							<CardTitle class="text-base capitalize">{p.platform}</CardTitle>
						</div>
						<CardDescription>{p.posts} post{p.posts === 1 ? '' : 's'}</CardDescription>
					</CardHeader>
					<CardContent>
						<dl class="grid grid-cols-2 gap-x-4 gap-y-2 text-sm">
							<div>
								<dt class="text-muted-foreground">Impressions</dt>
								<dd class="font-semibold tabular-nums">{formatCompact(p.impressions)}</dd>
							</div>
							<div>
								<dt class="text-muted-foreground">Likes</dt>
								<dd class="font-semibold tabular-nums">{formatCompact(p.likes)}</dd>
							</div>
							<div>
								<dt class="text-muted-foreground">Reposts</dt>
								<dd class="font-semibold tabular-nums">{formatCompact(p.reposts)}</dd>
							</div>
							<div>
								<dt class="text-muted-foreground">Comments</dt>
								<dd class="font-semibold tabular-nums">{formatCompact(p.comments)}</dd>
							</div>
						</dl>
					</CardContent>
				</Card>
			{/each}
		</div>
	{/if}

	<!-- Posts table -->
	<Card>
		<CardHeader>
			<div class="flex items-start justify-between gap-4">
				<div>
					<CardTitle>Posts</CardTitle>
					<CardDescription>Individual social posts with engagement counts</CardDescription>
				</div>
				{#if platforms.length > 1}
					<div class="flex flex-wrap gap-1.5">
						<button
							onclick={() => (platformFilter = 'all')}
							class="rounded-full px-3 py-1 text-xs font-medium transition-colors
								{platformFilter === 'all'
								? 'bg-primary text-primary-foreground'
								: 'bg-muted text-muted-foreground hover:text-foreground'}"
						>
							All
						</button>
						{#each platforms as p (p)}
							<button
								onclick={() => (platformFilter = p)}
								class="rounded-full px-3 py-1 text-xs font-medium capitalize transition-colors
									{platformFilter === p
									? 'bg-primary text-primary-foreground'
									: 'bg-muted text-muted-foreground hover:text-foreground'}"
							>
								{p}
							</button>
						{/each}
					</div>
				{/if}
			</div>
		</CardHeader>
		<CardContent class="p-0">
			{#if loading && !filtered.length}
				<div class="space-y-2 p-4">
					{#each Array(6) as _, i (i)}
						<Skeleton class="h-10 w-full" />
					{/each}
				</div>
			{:else if !filtered.length}
				<div class="text-muted-foreground p-8 text-center text-sm">No posts found</div>
			{:else}
				<Table>
					<THead>
						<TR>
							<TH>Post</TH>
							<TH>Platform</TH>
							<TH>Kind</TH>
							<TH class="text-right">Impressions</TH>
							<TH class="text-right">Likes</TH>
							<TH class="text-right">Reposts</TH>
							<TH class="text-right">Comments</TH>
							<TH>Date</TH>
						</TR>
					</THead>
					<TBody>
						{#each filtered as p (p.id)}
							<TR>
								<TD>
									{#if p.post_url}
										<a
											href={p.post_url}
											target="_blank"
											rel="noopener"
											class="hover:text-primary block max-w-[280px] truncate font-medium text-sm"
											title={p.text}
										>
											{p.text}
										</a>
									{:else}
										<span class="block max-w-[280px] truncate text-sm" title={p.text}>{p.text}</span>
									{/if}
								</TD>
								<TD>
									<div class="flex items-center gap-1.5">
										<span class="inline-block h-2 w-2 rounded-full {platformColor(p.platform)}"></span>
										<span class="capitalize text-sm">{p.platform}</span>
									</div>
								</TD>
								<TD>
									<Badge variant={kindVariant(p.kind)}>{kindLabel(p.kind)}</Badge>
								</TD>
								<TD class="text-right tabular-nums">{formatCompact(p.impressions)}</TD>
								<TD class="text-right tabular-nums">{formatCompact(p.likes)}</TD>
								<TD class="text-right tabular-nums">{formatCompact(p.reposts)}</TD>
								<TD class="text-right tabular-nums">{formatCompact(p.comments)}</TD>
								<TD class="text-muted-foreground text-xs">{formatDate(p.posted_at)}</TD>
							</TR>
						{/each}
					</TBody>
				</Table>
			{/if}
		</CardContent>
	</Card>
</div>
