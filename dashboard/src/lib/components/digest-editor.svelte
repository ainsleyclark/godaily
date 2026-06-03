<script lang="ts">
	import { untrack } from 'svelte';
	import type { DigestItem } from '$lib/api/types';
	import { groupBySection, SECTION_ORDER, type SectionTag, type Section } from '$lib/digest/sections';
	import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Badge } from '$lib/components/ui/badge';
	import { AlertDialog } from '$lib/components/ui/alert-dialog';
	import { DropdownMenu, DropdownMenuItem } from '$lib/components/ui/dropdown-menu';
	import { ExternalLink, GripVertical, Trash2, X } from '@lucide/svelte';
	import { dndzone, SHADOW_ITEM_MARKER_PROPERTY_NAME, type DndEvent } from 'svelte-dnd-action';
	import { flip } from 'svelte/animate';

	interface Props {
		items: DigestItem[];
		onReorder: (orderedIds: number[]) => void | Promise<void>;
		/** Unlinks the item from the issue, keeping it in the candidate pool. */
		onDelete: (itemId: number) => void | Promise<void>;
		/** Permanently deletes the item row from the database. */
		onHardDelete: (itemId: number) => void | Promise<void>;
		busy?: boolean;
	}

	let { items, onReorder, onDelete, onHardDelete, busy = false }: Props = $props();

	const FLIP_DURATION = 180;

	// svelte-dnd-action mutates and re-tracks the arrays it is given, so the
	// sections must be owned local state — deriving them straight from `items`
	// (the previous approach) swapped the arrays out from under the library
	// mid-drag and threw "Cannot read properties of undefined (parentElement)".
	// We resync from the prop whenever it changes, but never while a drag is in
	// flight.
	// Seed with the initial grouping (the $effect below keeps it in sync with
	// the prop); intentionally captures only the initial value.
	// svelte-ignore state_referenced_locally
	let sections = $state<Section[]>(groupBySection(items ?? []));
	let dragging = $state(false);
	// A single shared flag gates every zone: the whole card stays undraggable
	// until a grip handle is pressed, so the rest of the card scrolls/clicks
	// normally (essential on touch). Reset after each drop.
	let dragDisabled = $state(true);

	$effect(() => {
		const next = groupBySection(items ?? []);
		untrack(() => {
			if (!dragging) sections = next;
		});
	});

	// Running 1-based number for the first item of each section, so the badges
	// read as a single sequence down the whole digest (matching send order).
	const sectionOffsets = $derived.by(() => {
		const offsets: number[] = [];
		let running = 0;
		for (const section of sections) {
			offsets.push(running);
			running += section.items.length;
		}
		return offsets;
	});

	function isShadow(item: DigestItem): boolean {
		return Boolean((item as Record<string, unknown>)[SHADOW_ITEM_MARKER_PROPERTY_NAME]);
	}

	function startDrag(e: PointerEvent) {
		// Left button / touch only; arm dragging for the zone this handle is in.
		if (busy || (e.pointerType === 'mouse' && e.button !== 0)) return;
		e.preventDefault();
		dragging = true;
		dragDisabled = false;
	}

	function startKeyboardDrag(e: KeyboardEvent) {
		if (busy) return;
		if ((e.key === 'Enter' || e.key === ' ') && dragDisabled) {
			dragging = true;
			dragDisabled = false;
		}
	}

	function handleConsider(tag: SectionTag, e: CustomEvent<DndEvent<DigestItem>>) {
		const section = sections.find((s) => s.tag === tag);
		if (section) section.items = e.detail.items;
	}

	function handleFinalize(tag: SectionTag, e: CustomEvent<DndEvent<DigestItem>>) {
		const section = sections.find((s) => s.tag === tag);
		if (section) section.items = e.detail.items;
		dragging = false;
		dragDisabled = true;

		// Compose the full ordering across all sections in canonical order — the
		// API treats positions as one flat sequence, sections derived from tags.
		const ordered: number[] = [];
		for (const tagKey of SECTION_ORDER) {
			const s = sections.find((x) => x.tag === tagKey);
			if (!s) continue;
			for (const item of s.items) ordered.push(item.id);
		}
		void onReorder(ordered);
	}

	// The item awaiting a permanent-delete confirmation, or null when closed.
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
</script>

{#if sections.length === 0}
	<div class="text-muted-foreground p-8 text-center text-sm">No items in this issue yet</div>
{:else}
	<p class="text-muted-foreground mb-4 text-xs">
		This is exactly what sends, in order. Drag the <GripVertical class="inline h-3 w-3 align-text-bottom" />
		handle to reorder within a section.
	</p>
	<div class="space-y-4">
		{#each sections as section, si (section.tag)}
			<Card>
				<CardHeader>
					<CardTitle class="flex items-center gap-2">
						{section.title}
						<Badge variant="secondary">{section.items.length}</Badge>
					</CardTitle>
				</CardHeader>
				<CardContent>
					<ul
						class="space-y-2 rounded-md transition-colors"
						use:dndzone={{
							items: section.items,
							type: `digest-section-${section.tag}`,
							dragDisabled,
							flipDurationMs: FLIP_DURATION,
							dropTargetStyle: {},
							dropTargetClasses: ['ring-2', 'ring-primary/30', 'bg-primary/5']
						}}
						onconsider={(e) => handleConsider(section.tag, e)}
						onfinalize={(e) => handleFinalize(section.tag, e)}
					>
						{#each section.items as item, i (item.id)}
							<li
								animate:flip={{ duration: FLIP_DURATION }}
								class="group bg-card flex items-start gap-2 rounded-md border p-3 transition-shadow"
								class:opacity-40={isShadow(item)}
								class:border-dashed={isShadow(item)}
								class:shadow-md={!isShadow(item)}
							>
								<span
									class="bg-secondary text-muted-foreground mt-0.5 inline-flex h-5 min-w-5 shrink-0 items-center justify-center rounded px-1 text-xs font-medium tabular-nums"
									aria-hidden="true"
								>
									{sectionOffsets[si] + i + 1}
								</span>
								<button
									type="button"
									onpointerdown={startDrag}
									onkeydown={startKeyboardDrag}
									disabled={busy}
									class="text-muted-foreground hover:text-foreground mt-0.5 shrink-0 cursor-grab touch-none select-none active:cursor-grabbing disabled:cursor-not-allowed disabled:opacity-40"
									aria-label="Drag to reorder"
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
								<div class="shrink-0">
									<DropdownMenu label="Item actions" disabled={busy}>
										{#snippet children(close)}
											<DropdownMenuItem
												onclick={() => {
													void onDelete(item.id);
													close();
												}}
											>
												<X class="h-4 w-4" /> Remove from digest
											</DropdownMenuItem>
											<DropdownMenuItem
												variant="destructive"
												onclick={() => {
													requestHardDelete(item);
													close();
												}}
											>
												<Trash2 class="h-4 w-4" /> Delete permanently
											</DropdownMenuItem>
										{/snippet}
									</DropdownMenu>
								</div>
							</li>
						{/each}
					</ul>
				</CardContent>
			</Card>
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
	{busy}
	onConfirm={confirmHardDelete}
	onCancel={() => (pendingDelete = null)}
/>
