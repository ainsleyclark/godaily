<script lang="ts">
	import { page } from '$app/state';
	import { api, ApiError } from '$lib/api/client';
	import type { DigestIssue } from '$lib/api/types';
	import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';
	import { Button } from '$lib/components/ui/button';
	import { Badge } from '$lib/components/ui/badge';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import DigestPreview from '$lib/components/digest-preview.svelte';
	import DigestEditor from '$lib/components/digest-editor.svelte';
	import { ArrowLeft, ExternalLink } from '@lucide/svelte';
	import { formatDate } from '$lib/utils/format';
	import { toast } from 'svelte-sonner';

	const issueId = $derived(Number(page.params.id));

	let issue = $state<DigestIssue | null>(null);
	let loading = $state(true);
	let saving = $state(false);
	let mutatingItems = $state(false);

	let subject = $state('');
	let summary = $state('');

	const isDraft = $derived(issue?.status === 'draft');
	const dirty = $derived(
		issue !== null && (subject !== issue.subject || summary !== (issue.summary ?? ''))
	);

	async function load() {
		loading = true;
		try {
			const data = await api.digestIssueById(issueId);
			issue = data;
			subject = data.subject;
			summary = data.summary ?? '';
		} catch (e) {
			if ((e as ApiError).status !== 401) {
				toast.error((e as Error).message || 'Failed to load issue');
			}
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		if (Number.isFinite(issueId)) void load();
	});

	async function save() {
		if (!issue || !isDraft) return;
		saving = true;
		try {
			const updated = await api.updateDigestIssue(issue.id, {
				subject: subject.trim(),
				summary: summary.trim()
			});
			issue = updated;
			subject = updated.subject;
			summary = updated.summary ?? '';
			toast.success('Issue updated');
		} catch (e) {
			const msg = (e as Error).message || 'Failed to update issue';
			toast.error(msg);
		} finally {
			saving = false;
		}
	}

	function statusVariant(status: string): 'default' | 'secondary' | 'success' | 'destructive' {
		if (status === 'sent') return 'success';
		if (status === 'error') return 'destructive';
		return 'secondary';
	}

	async function reorderItems(orderedIds: number[]) {
		if (!issue || !isDraft || mutatingItems) return;
		const snapshot = issue;
		mutatingItems = true;
		try {
			issue = await api.reorderDigestItems(snapshot.id, orderedIds);
		} catch (e) {
			issue = snapshot;
			toast.error((e as Error).message || 'Failed to reorder items');
		} finally {
			mutatingItems = false;
		}
	}

	async function deleteItem(itemId: number) {
		if (!issue || !isDraft || mutatingItems) return;
		const snapshot = issue;
		mutatingItems = true;
		try {
			issue = await api.deleteDigestItem(snapshot.id, itemId);
			toast.success('Item removed');
		} catch (e) {
			issue = snapshot;
			toast.error((e as Error).message || 'Failed to remove item');
		} finally {
			mutatingItems = false;
		}
	}
</script>

<svelte:head><title>{issue ? issue.slug : 'Issue'} | GoDaily</title></svelte:head>

<div class="space-y-6">
	<div class="flex items-center justify-between gap-4">
		<div class="flex items-center gap-3">
			<a href="/issues" class="text-muted-foreground hover:text-foreground flex items-center gap-1 text-sm">
				<ArrowLeft class="h-4 w-4" /> Issues
			</a>
			{#if issue}
				<Badge variant={statusVariant(issue.status)}>{issue.status}</Badge>
				<span class="text-muted-foreground text-sm">#{issue.id} · {issue.slug}</span>
				<span class="text-muted-foreground text-xs">{formatDate(issue.sent_at)}</span>
			{/if}
		</div>
		<div class="flex items-center gap-2">
			{#if issue && issue.status === 'sent'}
				<a
					href={`https://godaily.dev/issues/${issue.slug}`}
					target="_blank"
					rel="noopener"
					class="text-muted-foreground hover:text-foreground inline-flex items-center gap-1 text-sm"
				>
					View live <ExternalLink class="h-3 w-3" />
				</a>
			{/if}
			{#if isDraft}
				<Button onclick={save} disabled={!dirty || saving || !subject.trim()}>
					{saving ? 'Saving…' : 'Save'}
				</Button>
			{/if}
		</div>
	</div>

	{#if loading && !issue}
		<div class="space-y-4">
			<Skeleton class="h-32 w-full" />
			<Skeleton class="h-64 w-full" />
		</div>
	{:else if !issue}
		<div class="text-muted-foreground p-8 text-center text-sm">Issue not found</div>
	{:else}
		<Card>
			<CardHeader>
				<CardTitle>{isDraft ? 'Edit fields' : 'Issue fields'}</CardTitle>
			</CardHeader>
			<CardContent class="space-y-4">
				<div class="space-y-1.5">
					<label for="subject" class="text-sm font-medium">Subject</label>
					<Input
						id="subject"
						bind:value={subject}
						readonly={!isDraft}
						placeholder="Email subject and page title"
					/>
				</div>
				<div class="space-y-1.5">
					<label for="summary" class="text-sm font-medium">Summary</label>
					<textarea
						id="summary"
						bind:value={summary}
						readonly={!isDraft}
						rows="3"
						class="border-input bg-background placeholder:text-muted-foreground focus-visible:ring-ring flex w-full rounded-md border px-3 py-2 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 disabled:cursor-not-allowed disabled:opacity-50"
						placeholder="Optional intro paragraph"
					></textarea>
				</div>
				{#if !isDraft}
					<p class="text-muted-foreground text-xs">Only draft issues can be edited.</p>
				{/if}
			</CardContent>
		</Card>

		<Card>
			<CardHeader>
				<CardTitle>{isDraft ? 'Edit items' : 'Preview'} ({issue.items?.length ?? 0} items)</CardTitle>
			</CardHeader>
			<CardContent>
				{#if isDraft}
					<DigestEditor
						items={issue.items ?? []}
						busy={mutatingItems}
						onReorder={reorderItems}
						onDelete={deleteItem}
					/>
				{:else}
					<DigestPreview items={issue.items ?? []} />
				{/if}
			</CardContent>
		</Card>
	{/if}
</div>
