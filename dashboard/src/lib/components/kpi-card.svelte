<script lang="ts">
	import { Card, CardContent } from '$lib/components/ui/card';
	import { Skeleton } from '$lib/components/ui/skeleton';

	interface Props {
		label: string;
		value: string;
		sublabel?: string;
		loading?: boolean;
	}
	let { label, value, sublabel, loading = false }: Props = $props();
</script>

<Card>
	<CardContent class="p-5">
		<div class="text-muted-foreground text-xs font-medium uppercase tracking-wider">{label}</div>
		<div class="mt-2 flex items-baseline gap-2">
			{#if loading}
				<Skeleton class="h-8 w-24" />
			{:else}
				<span class="text-foreground text-3xl font-semibold tabular-nums">{value}</span>
			{/if}
		</div>
		{#if sublabel && !loading}
			<div class="text-muted-foreground mt-1 text-xs">{sublabel}</div>
		{/if}
	</CardContent>
</Card>
