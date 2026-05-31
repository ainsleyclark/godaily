<script lang="ts">
	import { goto } from '$app/navigation';
	import { api, ApiError } from '$lib/api/client';
	import { auth } from '$lib/stores/auth';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';

	let password = $state('');
	let error = $state<string | null>(null);
	let submitting = $state(false);

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		error = null;
		if (!password.trim()) {
			error = 'Enter your password.';
			return;
		}
		submitting = true;
		try {
			const { token } = await api.login(password.trim());
			auth.setSecret(token);
			await goto('/');
		} catch (err) {
			if (err instanceof ApiError && err.status === 401) {
				error = 'Invalid password.';
			} else {
				error = (err as Error).message || 'Failed to sign in.';
			}
		} finally {
			submitting = false;
		}
	}
</script>

<svelte:head><title>Login | GoDaily Analytics</title></svelte:head>

<div class="flex min-h-[calc(100vh-3rem)] items-center justify-center">
	<Card class="w-full max-w-sm">
		<CardHeader>
			<div class="mb-2 flex items-center gap-2">
				<img src="/favicon.png" alt="GoDaily" class="h-7 w-7 rounded-md" />
				<span class="text-sm font-semibold">GoDaily</span>
			</div>
			<CardTitle>Dashboard</CardTitle>
			<CardDescription>Enter your password to continue.</CardDescription>
		</CardHeader>
		<CardContent>
			<form onsubmit={submit} class="space-y-4">
				<Input
					type="password"
					placeholder="Password"
					bind:value={password}
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
