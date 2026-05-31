<script lang="ts">
	import { Card, CardContent } from '$lib/components/ui/card';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { ArrowUpRight, ArrowDownRight } from '@lucide/svelte';

	interface Props {
		label: string;
		value: string;
		sublabel?: string;
		loading?: boolean;
		size?: 'sm' | 'lg';
		delta?: { value: string; direction: 'up' | 'down' | 'flat' };
	}
	let { label, value, sublabel, loading = false, size = 'sm', delta }: Props = $props();

	const isLg = $derived(size === 'lg');

	const deltaColor = $derived(
		delta?.direction === 'up'
			? 'text-emerald-400 bg-emerald-500/10 border-emerald-500/30'
			: delta?.direction === 'down'
				? 'text-destructive bg-destructive/10 border-destructive/30'
				: 'text-muted-foreground bg-secondary/40 border-border'
	);
</script>

<Card>
	<CardContent class={isLg ? 'p-6' : 'p-5'}>
		<div class="text-muted-foreground text-xs font-medium uppercase tracking-wider">{label}</div>
		<div class="mt-2 flex items-baseline gap-3">
			{#if loading}
				<Skeleton class={isLg ? 'h-12 w-32' : 'h-8 w-24'} />
			{:else}
				<span
					class="text-foreground font-semibold tabular-nums"
					class:text-3xl={!isLg}
					class:text-4xl={isLg}
				>
					{value}
				</span>
				{#if delta && !loading}
					<span
						class={`inline-flex items-center gap-0.5 rounded-md border px-1.5 py-0.5 text-xs font-medium ${deltaColor}`}
					>
						{#if delta.direction === 'up'}
							<ArrowUpRight class="h-3 w-3" strokeWidth={2.5} />
						{:else if delta.direction === 'down'}
							<ArrowDownRight class="h-3 w-3" strokeWidth={2.5} />
						{/if}
						{delta.value}
					</span>
				{/if}
			{/if}
		</div>
		{#if sublabel && !loading}
			<div class="text-muted-foreground mt-1 text-xs">{sublabel}</div>
		{/if}
	</CardContent>
</Card>
