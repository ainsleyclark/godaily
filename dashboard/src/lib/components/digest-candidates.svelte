<script lang="ts">
	import type { DigestItem } from '$lib/api/types';
	import { SECTION_TITLES, sectionOf } from '$lib/digest/sections';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { AlertDialog } from '$lib/components/ui/alert-dialog';
	import { DropdownMenu, DropdownMenuItem } from '$lib/components/ui/dropdown-menu';
	import { ExternalLink, Plus, Trash2 } from '@lucide/svelte';

	interface Props {
		items: DigestItem[];
		/** Links the item into the issue. */
		onAdd: (itemId: number) => void | Promise<void>;
		/** Permanently deletes the item row from the database. */
		onHardDelete: (itemId: number) => void | Promise<void>;
		busy?: boolean;
	}

	let { items, onAdd, onHardDelete, busy = false }: Props = $props();

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

	function sectionTitle(tag: string): string {
		return SECTION_TITLES[sectionOf(tag)];
	}
</script>

{#if items.length === 0}
	<div class="text-muted-foreground p-8 text-center text-sm">
		Nothing spare — every collected item for this issue's window is in the digest.
	</div>
{:else}
	<p class="text-muted-foreground mb-4 text-xs">
		Collected items left out of this issue. <strong>Add to digest</strong> drops one into its section;
		it then appears above and ships with the issue.
	</p>
	<ul class="space-y-2">
		{#each items as item (item.id)}
			<li class="group bg-card flex items-start gap-3 rounded-md border p-3">
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
					<div class="text-muted-foreground flex flex-wrap items-center gap-2 text-xs">
						<Badge variant="secondary">{sectionTitle(item.tag)}</Badge>
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
				<div class="flex shrink-0 items-center gap-1">
					<Button size="sm" variant="outline" onclick={() => onAdd(item.id)} disabled={busy}>
						<Plus class="h-4 w-4" /> Add to digest
					</Button>
					<DropdownMenu label="More actions" disabled={busy}>
						{#snippet children(close)}
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
