<script lang="ts">
	import { cn } from '$lib/utils';
	import type { Snippet } from 'svelte';
	import type { HTMLAttributes } from 'svelte/elements';

	type Variant = 'default' | 'secondary' | 'outline' | 'success' | 'destructive';
	interface Props extends HTMLAttributes<HTMLSpanElement> {
		variant?: Variant;
		class?: string;
		children?: Snippet;
	}
	let { variant = 'default', class: className, children, ...rest }: Props = $props();

	const styles: Record<Variant, string> = {
		default: 'bg-primary text-primary-foreground',
		secondary: 'bg-secondary text-secondary-foreground',
		outline: 'border border-input',
		success: 'bg-emerald-500/15 text-emerald-400 border border-emerald-500/30',
		destructive: 'bg-destructive/15 text-destructive border border-destructive/30'
	};
</script>

<span
	class={cn(
		'inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium',
		styles[variant],
		className
	)}
	{...rest}
>
	{@render children?.()}
</span>
