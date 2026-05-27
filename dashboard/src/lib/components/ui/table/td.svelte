<script lang="ts">
	import { cn } from '$lib/utils';
	import type { Snippet } from 'svelte';
	let { class: className = '', children }: { class?: string; children?: Snippet } = $props();
</script>

<td class={cn('p-3 align-middle', className)}>{@render children?.()}</td>
