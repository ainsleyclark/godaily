<script lang="ts">
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import type { SourceMetrics } from '$lib/api/types';
	import { formatCompact } from '$lib/utils/format';

	interface Props {
		data: SourceMetrics[] | null;
		loading?: boolean;
		limit?: number;
	}
	let { data, loading = false, limit = 10 }: Props = $props();

	const rows = $derived((data ?? []).slice(0, limit));
	const max = $derived(rows.length ? Math.max(...rows.map((r) => r.clicks), 1) : 1);
</script>

<Card>
	<CardHeader>
		<CardTitle>Top sources</CardTitle>
		<CardDescription>Where clicks come from</CardDescription>
	</CardHeader>
	<CardContent>
		{#if loading && !rows.length}
			<div class="space-y-3">
				{#each Array(5) as _, i (i)}
					<Skeleton class="h-6 w-full" />
				{/each}
			</div>
		{:else if !rows.length}
			<div class="text-muted-foreground py-6 text-center text-sm">No source data</div>
		{:else}
			<div class="space-y-3">
				{#each rows as r (r.source)}
					<div class="space-y-1">
						<div class="flex items-center justify-between text-sm">
							<span class="text-foreground truncate font-medium" title={r.source}>{r.source}</span>
							<span class="text-muted-foreground tabular-nums">{formatCompact(r.clicks)}</span>
						</div>
						<div class="bg-secondary/40 h-1.5 w-full overflow-hidden rounded-full">
							<div
								class="h-full rounded-full"
								style="width:{(r.clicks / max) * 100}%; background:var(--chart-2)"
							></div>
						</div>
					</div>
				{/each}
			</div>
		{/if}
	</CardContent>
</Card>
