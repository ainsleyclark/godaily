<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { api } from '$lib/api/client';
	import type { SocialPost } from '$lib/api/types';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Separator } from '$lib/components/ui/separator';
	import { toast } from 'svelte-sonner';

	let drafts = $state<SocialPost[] | null>(null);
	let loading = $state(true);
	let edits = $state<Record<number, string>>({});
	let savingId = $state<number | null>(null);
	let cancellingId = $state<number | null>(null);

	// Per-platform character ceilings used by the UI hint. Mirrors the
	// platform.PostRequest constraints enforced server-side.
	const PLATFORM_LIMITS: Record<string, number> = {
		bluesky: 300,
		mastodon: 500,
		linkedin: 3000
	};

	const KIND_LABELS: Record<string, string> = {
		featured: 'Featured',
		recap: 'Recap',
		spotlight: 'Spotlight',
		cta: 'CTA',
		community: 'Community',
		new_source: 'New Source'
	};

	async function load() {
		loading = true;
		try {
			const rows = await api.socialDrafts();
			drafts = rows;
			// Seed the local edit buffer with the server values so the textarea
			// reflects unsaved-vs-saved by simple string comparison.
			const next: Record<number, string> = {};
			for (const d of rows) next[d.id] = d.text;
			edits = next;
		} catch (e) {
			if ((e as { status?: number }).status !== 401) {
				toast.error('Failed to load drafts');
			}
		} finally {
			loading = false;
		}
	}

	onMount(() => {
		void load();
	});

	// Scroll the deep-linked row into view once the list is rendered.
	$effect(() => {
		const id = Number(page.url.searchParams.get('id'));
		if (!id || !drafts) return;
		queueMicrotask(() => {
			const el = document.getElementById(`draft-${id}`);
			if (el) el.scrollIntoView({ behavior: 'smooth', block: 'center' });
		});
	});

	function kindLabel(kind: string) {
		return KIND_LABELS[kind] ?? kind;
	}

	function platformLimit(platform: string): number | null {
		return PLATFORM_LIMITS[platform.toLowerCase()] ?? null;
	}

	function isDirty(d: SocialPost): boolean {
		return (edits[d.id] ?? '') !== d.text;
	}

	async function save(d: SocialPost) {
		const text = (edits[d.id] ?? '').trim();
		if (!text) {
			toast.error('Text is required');
			return;
		}
		savingId = d.id;
		try {
			const updated = await api.updateSocialDraft(d.id, text);
			drafts = (drafts ?? []).map((row) => (row.id === d.id ? updated : row));
			edits = { ...edits, [d.id]: updated.text };
			toast.success('Draft updated');
		} catch (e) {
			toast.error(`Failed to update — ${(e as Error).message}`);
		} finally {
			savingId = null;
		}
	}

	async function cancel(d: SocialPost) {
		if (!confirm(`Cancel this ${kindLabel(d.kind)} · ${d.platform} draft? It won't be published.`)) return;
		cancellingId = d.id;
		try {
			await api.cancelSocialDraft(d.id);
			drafts = (drafts ?? []).filter((row) => row.id !== d.id);
			toast.success('Draft cancelled');
		} catch (e) {
			toast.error(`Failed to cancel — ${(e as Error).message}`);
		} finally {
			cancellingId = null;
		}
	}

	const deepLinkId = $derived(Number(page.url.searchParams.get('id')) || null);

	const grouped = $derived.by(() => {
		const rows = drafts ?? [];
		const byKind: Record<string, SocialPost[]> = {};
		for (const r of rows) {
			(byKind[r.kind] ??= []).push(r);
		}
		const order = ['featured', 'recap', 'community', 'spotlight', 'new_source', 'cta'];
		return Object.entries(byKind).sort((a, b) => order.indexOf(a[0]) - order.indexOf(b[0]));
	});
</script>

<svelte:head><title>Social drafts | GoDaily Analytics</title></svelte:head>

<div class="space-y-6">
	<div>
		<h1 class="text-xl font-semibold tracking-tight">Social drafts</h1>
		<p class="text-muted-foreground text-sm mt-1">
			Edit or cancel pending social posts before the 11:00 UTC publish cron picks them up.
		</p>
	</div>

	{#if loading && !drafts}
		<div class="space-y-2">
			{#each Array(3) as _, i (i)}
				<Skeleton class="h-32 w-full" />
			{/each}
		</div>
	{:else if !drafts || drafts.length === 0}
		<Card>
			<CardContent class="text-muted-foreground p-8 text-center text-sm">
				No pending drafts. Today's build cron will populate this list.
			</CardContent>
		</Card>
	{:else}
		{#each grouped as [kind, rows] (kind)}
			<Card>
				<CardHeader>
					<div class="flex items-center gap-2">
						<CardTitle>{kindLabel(kind)}</CardTitle>
						<Badge variant="secondary">{rows.length}</Badge>
					</div>
					<CardDescription>One row per platform. Publishes 11:00 UTC unless cancelled.</CardDescription>
				</CardHeader>
				<CardContent class="space-y-4">
					{#each rows as d, i (d.id)}
						{@const limit = platformLimit(d.platform)}
						{@const length = (edits[d.id] ?? '').length}
						{@const over = limit !== null && length > limit}
						<div
							id={`draft-${d.id}`}
							class="rounded-md border p-4 transition-colors {deepLinkId === d.id ? 'border-primary' : ''}"
						>
							<div class="flex items-center justify-between gap-2 mb-2">
								<div class="flex items-center gap-2">
									<Badge class="capitalize">{d.platform}</Badge>
									{#if d.subject}<span class="text-muted-foreground text-xs">{d.subject}</span>{/if}
								</div>
								<span class="text-muted-foreground text-xs tabular-nums {over ? 'text-red-500' : ''}">
									{length}{limit !== null ? ` / ${limit}` : ''} chars
								</span>
							</div>
							<textarea
								bind:value={edits[d.id]}
								rows="6"
								class="border-input bg-background w-full rounded-md border px-3 py-2 text-sm font-mono"
								placeholder="Empty draft"
							></textarea>
							<div class="flex items-center justify-end gap-2 mt-2">
								<Button
									variant="outline"
									size="sm"
									disabled={cancellingId === d.id || savingId === d.id}
									onclick={() => cancel(d)}
								>
									{cancellingId === d.id ? 'Cancelling…' : 'Cancel draft'}
								</Button>
								<Button
									size="sm"
									disabled={!isDirty(d) || savingId === d.id || cancellingId === d.id}
									onclick={() => save(d)}
								>
									{savingId === d.id ? 'Saving…' : 'Save'}
								</Button>
							</div>
						</div>
						{#if i < rows.length - 1}<Separator />{/if}
					{/each}
				</CardContent>
			</Card>
		{/each}
	{/if}
</div>
