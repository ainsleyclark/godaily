<script lang="ts">
	import { cn } from '$lib/utils';
	import { EllipsisVertical } from '@lucide/svelte';
	import type { Snippet } from 'svelte';

	interface Props {
		/** Menu contents. Receives a `close` callback to dismiss after an action. */
		children: Snippet<[() => void]>;
		/** Accessible label for the trigger button. */
		label?: string;
		/** Which edge of the trigger the panel aligns to. */
		align?: 'start' | 'end';
		disabled?: boolean;
		class?: string;
	}

	let { children, label = 'Open actions menu', align = 'end', disabled = false, class: className }: Props =
		$props();

	let open = $state(false);
	let root = $state<HTMLDivElement>();

	function close() {
		open = false;
	}

	function toggle() {
		if (disabled) return;
		open = !open;
	}

	// Dismiss on any click outside the menu root, mirroring the alert-dialog
	// overlay pattern. Only attached while open.
	function onWindowPointerDown(e: PointerEvent) {
		if (root && !root.contains(e.target as Node)) close();
	}

	function onWindowKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') close();
	}
</script>

<svelte:window
	onpointerdown={open ? onWindowPointerDown : undefined}
	onkeydown={open ? onWindowKeydown : undefined}
/>

<div bind:this={root} class={cn('relative', className)}>
	<button
		type="button"
		onclick={toggle}
		{disabled}
		aria-haspopup="menu"
		aria-expanded={open}
		aria-label={label}
		class="text-muted-foreground hover:text-foreground hover:bg-accent inline-flex h-8 w-8 items-center justify-center rounded-md transition-colors disabled:pointer-events-none disabled:opacity-50"
	>
		<EllipsisVertical class="h-4 w-4" />
	</button>

	{#if open}
		<div
			role="menu"
			class={cn(
				'bg-popover text-popover-foreground absolute z-50 mt-1 min-w-[12rem] overflow-hidden rounded-md border p-1 shadow-md',
				align === 'end' ? 'right-0' : 'left-0'
			)}
		>
			{@render children(close)}
		</div>
	{/if}
</div>
