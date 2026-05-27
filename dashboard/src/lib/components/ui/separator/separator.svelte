<script lang="ts">
	import { cn } from '$lib/utils';
	let {
		class: className = '',
		orientation = 'horizontal'
	}: { class?: string; orientation?: 'horizontal' | 'vertical' } = $props();
</script>

<div
	role="separator"
	class={cn(
		'shrink-0 bg-border',
		orientation === 'horizontal' ? 'h-px w-full' : 'h-full w-px',
		className
	)}
></div>
