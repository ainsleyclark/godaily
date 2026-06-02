<script lang="ts">
	import type { DigestItem } from '$lib/api/types';
	import { groupBySection } from '$lib/digest/sections';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { AlertDialog } from '$lib/components/ui/alert-dialog';
	import { ExternalLink, Trash2 } from '@lucide/svelte';

	interface Props {
		items: DigestItem[];
		/**
		 * When provided, each item gains a permanent-delete action. Omit to keep
		 * the preview purely read-only.
		 */
		onHardDelete?: (itemId: number) => void | Promise<void>;
		/** True while a delete is in flight; disables the action. */
		busy?: boolean;
	}
	let { items, onHardDelete, busy = false }: Props = $props();

	const sections = $derived(groupBySection(items ?? []));

	// The item awaiting a permanent-delete confirmation, or null when closed.
	let pendingDelete = $state<DigestItem | null>(null);

	function requestHardDelete(item: DigestItem) {
		if (busy) return;
		pendingDelete = item;
	}

	async function confirmHardDelete() {
		const item = pendingDelete;
		if (!item || !onHardDelete) return;
		await onHardDelete(item.id);
		pendingDelete = null;
	}
</script>

{#if sections.length === 0}
	<div class="text-muted-foreground p-8 text-center text-sm">No items in this issue</div>
{:else}
	<div class="space-y-8">
		{#each sections as section (section.tag)}
			<section>
				<header class="mb-3 flex items-center gap-2 border-b pb-2">
					<h3 class="text-base font-semibold tracking-tight">{section.title}</h3>
					<Badge variant="secondary">{section.items.length}</Badge>
				</header>
				<ul class="space-y-4">
					{#each section.items as item (item.id)}
						<li class="group flex items-start gap-2 space-y-1">
							<div class="min-w-0 flex-1 space-y-1">
								<div class="flex items-baseline gap-2">
									<a
										href={item.url}
										target="_blank"
										rel="noopener"
										class="hover:text-primary text-sm font-medium leading-snug"
									>
										{item.title}
									</a>
									<ExternalLink class="text-muted-foreground h-3 w-3 shrink-0" />
								</div>
								<div class="text-muted-foreground flex items-center gap-2 text-xs">
									<span class="capitalize">{item.source}</span>
									{#if item.author?.name || item.author?.username}
										<span>·</span>
										<span>{item.author.name || item.author.username}</span>
									{/if}
								</div>
								{#if item.snippet}
									<p class="text-muted-foreground text-sm leading-relaxed">{item.snippet}</p>
								{/if}
							</div>
							{#if onHardDelete}
								<Button
									variant="ghost"
									size="icon"
									onclick={() => requestHardDelete(item)}
									disabled={busy}
									aria-label="Delete permanently"
									title="Delete permanently from the database"
									class="text-muted-foreground hover:text-destructive shrink-0 opacity-0 transition group-hover:opacity-100 pointer-coarse:opacity-100"
								>
									<Trash2 class="h-4 w-4" />
								</Button>
							{/if}
						</li>
					{/each}
				</ul>
			</section>
		{/each}
	</div>
{/if}

{#if onHardDelete}
	<AlertDialog
		open={pendingDelete !== null}
		title="Delete item permanently?"
		description={pendingDelete
			? `"${pendingDelete.title}" will be removed from the database and cannot be recovered. It will not reappear in a future build.`
			: ''}
		confirmLabel="Delete permanently"
		busy={busy}
		onConfirm={confirmHardDelete}
		onCancel={() => (pendingDelete = null)}
	/>
{/if}
