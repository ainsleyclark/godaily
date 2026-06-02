<script lang="ts">
	import { Button, type Variant } from '$lib/components/ui/button';

	interface Props {
		/** Controls visibility. Bindable so callers can close on cancel/confirm. */
		open?: boolean;
		title: string;
		description?: string;
		confirmLabel?: string;
		cancelLabel?: string;
		confirmVariant?: Variant;
		/** True while the confirm action is in flight; disables the buttons. */
		busy?: boolean;
		onConfirm?: () => void | Promise<void>;
		onCancel?: () => void;
	}

	let {
		open = $bindable(false),
		title,
		description,
		confirmLabel = 'Confirm',
		cancelLabel = 'Cancel',
		confirmVariant = 'destructive',
		busy = false,
		onConfirm,
		onCancel
	}: Props = $props();

	function cancel() {
		if (busy) return;
		open = false;
		onCancel?.();
	}

	function confirm() {
		void onConfirm?.();
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') cancel();
	}
</script>

<svelte:window onkeydown={open ? handleKeydown : undefined} />

{#if open}
	<!-- Overlay: clicking outside the panel cancels. -->
	<div
		class="bg-background/80 fixed inset-0 z-50 flex items-center justify-center p-4 backdrop-blur-sm"
		role="presentation"
		onclick={cancel}
	>
		<!-- Panel: stop propagation so inner clicks don't dismiss. -->
		<div
			class="bg-background w-full max-w-md rounded-lg border p-6 shadow-lg"
			role="alertdialog"
			aria-modal="true"
			aria-labelledby="alert-dialog-title"
			aria-describedby={description ? 'alert-dialog-description' : undefined}
			tabindex="-1"
			onclick={(e) => e.stopPropagation()}
			onkeydown={(e) => e.stopPropagation()}
		>
			<h2 id="alert-dialog-title" class="text-lg font-semibold">{title}</h2>
			{#if description}
				<p id="alert-dialog-description" class="text-muted-foreground mt-2 text-sm">
					{description}
				</p>
			{/if}
			<div class="mt-6 flex justify-end gap-2">
				<Button variant="outline" onclick={cancel} disabled={busy}>{cancelLabel}</Button>
				<Button variant={confirmVariant} onclick={confirm} disabled={busy}>
					{busy ? 'Working…' : confirmLabel}
				</Button>
			</div>
		</div>
	</div>
{/if}
