<script lang="ts">
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Table, THead, TBody, TR, TH, TD } from '$lib/components/ui/table';
	import { Badge } from '$lib/components/ui/badge';
	import type { ItemMetrics } from '$lib/api/types';
	import { formatCompact } from '$lib/utils/format';
	import { prettify } from '$lib/utils/labels';

	interface Props {
		data: ItemMetrics[] | null;
		loading?: boolean;
		limit?: number;
	}
	let { data, loading = false, limit = 8 }: Props = $props();

	const rows = $derived((data ?? []).slice(0, limit));
</script>

<Card>
	<CardHeader>
		<CardTitle>Top items</CardTitle>
		<CardDescription>Most clicked links</CardDescription>
	</CardHeader>
	<CardContent class="p-0">
		{#if loading && !rows.length}
			<div class="space-y-2 p-4">
				{#each Array(5) as _, i (i)}
					<Skeleton class="h-9 w-full" />
				{/each}
			</div>
		{:else if !rows.length}
			<div class="text-muted-foreground p-8 text-center text-sm">No clicks yet</div>
		{:else}
			<Table>
				<THead>
					<TR>
						<TH>Title</TH>
						<TH>Source</TH>
						<TH class="text-right">Clicks</TH>
					</TR>
				</THead>
				<TBody>
					{#each rows as r (r.item_id)}
						<TR>
							<TD>
								<a
									href={r.url}
									target="_blank"
									rel="noopener"
									class="hover:text-primary block max-w-[300px] truncate font-medium"
									title={r.title}
								>
									{r.title}
								</a>
								{#if r.tag}
									<Badge variant="outline" class="mt-1">{prettify(r.tag)}</Badge>
								{/if}
							</TD>
							<TD class="text-muted-foreground text-xs">{prettify(r.source)}</TD>
							<TD class="text-right tabular-nums font-medium">{formatCompact(r.clicks)}</TD>
						</TR>
					{/each}
				</TBody>
			</Table>
		{/if}
	</CardContent>
</Card>
