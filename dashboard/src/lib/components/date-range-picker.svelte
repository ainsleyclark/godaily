<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import { dateRange, presetRange, type RangePreset } from '$lib/stores/dateRange';
	import type { Bucket } from '$lib/api/types';

	const presets: { label: string; value: RangePreset }[] = [
		{ label: '7d', value: '7d' },
		{ label: '30d', value: '30d' },
		{ label: '90d', value: '90d' },
		{ label: 'YTD', value: 'ytd' }
	];

	const buckets: Bucket[] = ['day', 'week', 'month'];

	function selectPreset(p: RangePreset) {
		dateRange.set(presetRange(p));
	}

	function setBucket(b: Bucket) {
		dateRange.update((r) => ({ ...r, bucket: b }));
	}
</script>

<div class="flex items-center gap-1">
	<div class="bg-secondary/40 flex items-center rounded-md p-0.5">
		{#each presets as p (p.value)}
			<button
				type="button"
				onclick={() => selectPreset(p.value)}
				class="rounded px-2.5 py-1 text-xs font-medium transition-colors"
				class:bg-background={$dateRange.preset === p.value}
				class:text-foreground={$dateRange.preset === p.value}
				class:shadow-sm={$dateRange.preset === p.value}
				class:text-muted-foreground={$dateRange.preset !== p.value}
			>
				{p.label}
			</button>
		{/each}
	</div>
	<div class="bg-secondary/40 ml-2 flex items-center rounded-md p-0.5">
		{#each buckets as b (b)}
			<button
				type="button"
				onclick={() => setBucket(b)}
				class="rounded px-2.5 py-1 text-xs font-medium capitalize transition-colors"
				class:bg-background={$dateRange.bucket === b}
				class:text-foreground={$dateRange.bucket === b}
				class:shadow-sm={$dateRange.bucket === b}
				class:text-muted-foreground={$dateRange.bucket !== b}
			>
				{b}
			</button>
		{/each}
	</div>
</div>
