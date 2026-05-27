<script lang="ts">
	import { goto } from '$app/navigation';
	import { api, ApiError } from '$lib/api/client';
	import { auth } from '$lib/stores/auth';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';

	let secret = $state('');
	let error = $state<string | null>(null);
	let submitting = $state(false);

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		error = null;
		if (!secret.trim()) {
			error = 'Enter the API secret.';
			return;
		}
		submitting = true;
		try {
			console.log('ddd')
			await api.summary({}, secret.trim());
			auth.setSecret(secret.trim());
			await goto('/');
		} catch (err) {
			if (err instanceof ApiError && err.status === 401) {
				error = 'Invalid secret.';
			} else {
				error = (err as Error).message || 'Failed to verify secret.';
			}
		} finally {
			submitting = false;
		}
	}
</script>

<div class="flex min-h-[calc(100vh-3rem)] items-center justify-center">
	<Card class="w-full max-w-sm">
		<CardHeader>
			<div class="mb-2 flex items-center gap-2">
				<span
					class="inline-flex h-7 w-7 items-center justify-center rounded-md text-sm font-bold"
					style="background:var(--chart-1); color:var(--primary-foreground)"
				>
					G
				</span>
				<span class="text-sm font-semibold">GoDaily</span>
			</div>
			<CardTitle>Mission Control</CardTitle>
			<CardDescription>Enter the API secret to continue.</CardDescription>
		</CardHeader>
		<CardContent>
			<form onsubmit={submit} class="space-y-4">
				<Input
					type="password"
					placeholder="API secret"
					bind:value={secret}
					autocomplete="current-password"
					autofocus
				/>
				{#if error}
					<p class="text-destructive text-xs">{error}</p>
				{/if}
				<Button type="submit" class="w-full" disabled={submitting}>
					{submitting ? 'Verifying…' : 'Sign in'}
				</Button>
			</form>
		</CardContent>
	</Card>
</div>
