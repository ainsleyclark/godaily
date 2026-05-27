<script lang="ts">
	import { cn } from '$lib/utils';
	import type { Snippet } from 'svelte';
	let { class: className = '', children }: { class?: string; children?: Snippet } = $props();
</script>

<tr class={cn('border-b transition-colors hover:bg-muted/40', className)}>{@render children?.()}</tr>
