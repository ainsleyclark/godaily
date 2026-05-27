<script lang="ts">
	import { cn } from '$lib/utils';
	import type { Snippet } from 'svelte';
	let { class: className = '', children }: { class?: string; children?: Snippet } = $props();
</script>

<th
	class={cn(
		'h-10 px-3 text-left align-middle text-xs font-medium text-muted-foreground uppercase tracking-wide',
		className
	)}
>
	{@render children?.()}
</th>
