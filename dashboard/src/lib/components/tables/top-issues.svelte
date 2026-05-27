<script lang="ts">
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Table, THead, TBody, TR, TH, TD } from '$lib/components/ui/table';
	import { Badge } from '$lib/components/ui/badge';
	import type { IssueEngagement } from '$lib/api/types';
	import { formatCompact, formatPercent, formatDate } from '$lib/utils/format';

	interface Props {
		data: IssueEngagement[] | null;
		loading?: boolean;
		limit?: number;
	}
	let { data, loading = false, limit = 8 }: Props = $props();

	const rows = $derived(
		(data ?? [])
			.slice()
			.sort((a, b) => b.unique_clicks - a.unique_clicks)
			.slice(0, limit)
	);
</script>

<Card>
	<CardHeader>
		<CardTitle>Top issues</CardTitle>
		<CardDescription>Ranked by unique clicks</CardDescription>
	</CardHeader>
	<CardContent class="p-0">
		{#if loading && !rows.length}
			<div class="space-y-2 p-4">
				{#each Array(5) as _, i (i)}
					<Skeleton class="h-9 w-full" />
				{/each}
			</div>
		{:else if !rows.length}
			<div class="text-muted-foreground p-8 text-center text-sm">No issues yet</div>
		{:else}
			<Table>
				<THead>
					<TR>
						<TH>Issue</TH>
						<TH class="text-right">Delivered</TH>
						<TH class="text-right">Open</TH>
						<TH class="text-right">Click</TH>
					</TR>
				</THead>
				<TBody>
					{#each rows as r (r.issue_id)}
						<TR>
							<TD>
								<a
									href={`https://godaily.dev/issues/${r.slug}`}
									target="_blank"
									rel="noopener"
									class="hover:text-primary block max-w-[260px] truncate font-medium"
									title={r.slug}
								>
									{r.slug}
								</a>
								<div class="text-muted-foreground text-xs">{formatDate(r.sent_at)}</div>
							</TD>
							<TD class="text-right tabular-nums">{formatCompact(r.delivered)}</TD>
							<TD class="text-right tabular-nums">
								<Badge variant="secondary">{formatPercent(r.open_rate)}</Badge>
							</TD>
							<TD class="text-right tabular-nums">
								<Badge variant="success">{formatPercent(r.click_rate)}</Badge>
							</TD>
						</TR>
					{/each}
				</TBody>
			</Table>
		{/if}
	</CardContent>
</Card>
