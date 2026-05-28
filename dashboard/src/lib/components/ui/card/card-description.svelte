<script lang="ts">
	import { cn } from '$lib/utils';
	import type { Snippet } from 'svelte';
	let { class: className = '', children }: { class?: string; children?: Snippet } = $props();
</script>

<p class={cn('text-muted-foreground text-sm', className)}>
	{@render children?.()}
</p>
