<script lang="ts">
	import { cn } from '$lib/utils';
	import type { Snippet } from 'svelte';
	import type { HTMLButtonAttributes } from 'svelte/elements';

	interface Props extends HTMLButtonAttributes {
		variant?: 'default' | 'destructive';
		class?: string;
		children?: Snippet;
	}

	let { variant = 'default', class: className, children, type = 'button', ...rest }: Props = $props();
</script>

<button
	{type}
	role="menuitem"
	class={cn(
		'flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-left text-sm transition-colors disabled:pointer-events-none disabled:opacity-50',
		variant === 'destructive'
			? 'text-destructive hover:bg-destructive/10'
			: 'hover:bg-accent hover:text-accent-foreground',
		className
	)}
	{...rest}
>
	{@render children?.()}
</button>
