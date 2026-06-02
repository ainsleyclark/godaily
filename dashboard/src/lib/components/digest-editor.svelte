<script lang="ts">
	import type { DigestItem } from '$lib/api/types';
	import { groupBySection, SECTION_ORDER, type SectionTag } from '$lib/digest/sections';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { AlertDialog } from '$lib/components/ui/alert-dialog';
	import { ExternalLink, GripVertical, Trash2, X } from '@lucide/svelte';
	import { dndzone, type DndEvent } from 'svelte-dnd-action';
	import { flip } from 'svelte/animate';

	interface Props {
		items: DigestItem[];
		onReorder: (orderedIds: number[]) => void | Promise<void>;
		/** Unlinks the item from the issue, keeping it in the raw pool. */
		onDelete: (itemId: number) => void | Promise<void>;
		/** Permanently deletes the item row from the database. */
		onHardDelete: (itemId: number) => void | Promise<void>;
		busy?: boolean;
	}

	let { items, onReorder, onDelete, onHardDelete, busy = false }: Props = $props();

	// The item awaiting a permanent-delete confirmation, or null when the dialog
	// is closed.
	let pendingDelete = $state<DigestItem | null>(null);

	function requestHardDelete(item: DigestItem) {
		if (busy) return;
		pendingDelete = item;
	}

	async function confirmHardDelete() {
		const item = pendingDelete;
		if (!item) return;
		await onHardDelete(item.id);
		pendingDelete = null;
	}

	// dnd-action mutates the array it operates on, so each section gets its own
	// local copy keyed off the parent items prop. Re-derive whenever items change.
	let sections = $derived(groupBySection(items ?? []));

	const FLIP_DURATION = 180;

	function handleConsider(tag: SectionTag, e: CustomEvent<DndEvent<DigestItem>>) {
		// Optimistically reflect the in-flight order while the user is dragging,
		// without committing to the server. Mutate the local section copy in place.
		const section = sections.find((s) => s.tag === tag);
		if (section) section.items = e.detail.items;
		sections = sections;
	}

	function handleFinalize(tag: SectionTag, e: CustomEvent<DndEvent<DigestItem>>) {
		const section = sections.find((s) => s.tag === tag);
		if (!section) return;
		section.items = e.detail.items;
		sections = sections;

		// Compose the full ordering across all sections in canonical SECTION_ORDER
		// — the API treats positions as a single flat sequence, with sections
		// derived from item tags.
		const ordered: number[] = [];
		for (const tagKey of SECTION_ORDER) {
			const s = sections.find((x) => x.tag === tagKey);
			if (!s) continue;
			for (const item of s.items) ordered.push(item.id);
		}
		void onReorder(ordered);
	}

	function confirmDelete(item: DigestItem) {
		if (busy) return;
		if (!confirm(`Remove "${item.title}" from this issue?`)) return;
		void onDelete(item.id);
	}
</script>

{#if sections.length === 0}
	<div class="text-muted-foreground p-8 text-center text-sm">No items in this issue</div>
{:else}
	<p class="text-muted-foreground mb-4 text-xs">
		Drag to reorder within a section. <strong>Remove from issue</strong> unlinks an item but keeps
		it in the raw pool; <strong>Delete permanently</strong> removes it from the database for good.
	</p>
	<div class="space-y-8">
		{#each sections as section (section.tag)}
			<section>
				<header class="mb-3 flex items-center gap-2 border-b pb-2">
					<h3 class="text-base font-semibold tracking-tight">{section.title}</h3>
					<Badge variant="secondary">{section.items.length}</Badge>
				</header>
				<ul
					class="space-y-2"
					use:dndzone={{
						items: section.items,
						type: `digest-section-${section.tag}`,
						dropTargetStyle: {},
						flipDurationMs: FLIP_DURATION,
						dragDisabled: busy
					}}
					onconsider={(e) => handleConsider(section.tag, e)}
					onfinalize={(e) => handleFinalize(section.tag, e)}
				>
					{#each section.items as item (item.id)}
						<li
							animate:flip={{ duration: FLIP_DURATION }}
							class="group bg-card hover:border-foreground/20 flex items-start gap-2 rounded-md border p-3"
						>
							<button
								type="button"
								class="text-muted-foreground hover:text-foreground mt-0.5 cursor-grab touch-none active:cursor-grabbing"
								aria-label="Drag to reorder"
								tabindex="-1"
							>
								<GripVertical class="h-4 w-4" />
							</button>
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
							<div
								class="flex shrink-0 items-center gap-0.5 opacity-0 transition group-hover:opacity-100"
							>
								<Button
									variant="ghost"
									size="icon"
									onclick={() => confirmDelete(item)}
									disabled={busy}
									aria-label="Remove from issue"
									title="Remove from issue (keeps in pool)"
									class="text-muted-foreground hover:text-foreground"
								>
									<X class="h-4 w-4" />
								</Button>
								<Button
									variant="ghost"
									size="icon"
									onclick={() => requestHardDelete(item)}
									disabled={busy}
									aria-label="Delete permanently"
									title="Delete permanently from the database"
									class="text-muted-foreground hover:text-destructive"
								>
									<Trash2 class="h-4 w-4" />
								</Button>
							</div>
						</li>
					{/each}
				</ul>
			</section>
		{/each}
	</div>
{/if}

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
