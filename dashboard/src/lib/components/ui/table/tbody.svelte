<script lang="ts">
	import { cn } from '$lib/utils';
	import type { Snippet } from 'svelte';
	let { class: className = '', children }: { class?: string; children?: Snippet } = $props();
</script>

<tbody class={cn('[&_tr:last-child]:border-0', className)}>{@render children?.()}</tbody>
