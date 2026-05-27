<script lang="ts">
	import '../app.css';
	import '../hooks.client';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { Button } from '$lib/components/ui/button';
	import DateRangePicker from '$lib/components/date-range-picker.svelte';
	import ThemeToggle from '$lib/components/theme-toggle.svelte';
	import { auth } from '$lib/stores/auth';
	import { Toaster } from 'svelte-sonner';

	let { children } = $props();

	const isLogin = $derived($page.url.pathname === '/login');

	let mobileOpen = $state(false);

	function logout() {
		auth.clearSecret();
		void goto('/login');
	}
</script>

<Toaster theme="dark" position="top-right" richColors />

<div class="min-h-screen">
	{#if !isLogin}
		<header class="border-border/60 bg-background/80 sticky top-0 z-40 border-b backdrop-blur">
			<div class="mx-auto flex h-14 max-w-[1400px] items-center gap-4 px-4 sm:px-6">
				<div class="flex items-center gap-2">
					<img src="/favicon.png" alt="GoDaily" class="h-7 w-7 rounded-md" />
					<span class="text-sm font-semibold tracking-tight">GoDaily Dashboard</span>
				</div>

				<!-- Desktop controls -->
				<div class="ml-auto hidden items-center gap-2 md:flex">
					<DateRangePicker />
					<div class="bg-border mx-1 h-6 w-px"></div>
					<ThemeToggle />
					<Button variant="ghost" size="sm" onclick={logout}>Logout</Button>
				</div>

				<!-- Mobile toggle -->
				<button
					type="button"
					class="ml-auto inline-flex h-9 w-9 items-center justify-center rounded-md md:hidden hover:bg-accent"
					aria-label="Toggle menu"
					aria-expanded={mobileOpen}
					onclick={() => (mobileOpen = !mobileOpen)}
				>
					{#if mobileOpen}
						<svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<path d="M18 6L6 18M6 6l12 12" />
						</svg>
					{:else}
						<svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<path d="M3 6h18M3 12h18M3 18h18" />
						</svg>
					{/if}
				</button>
			</div>

			{#if mobileOpen}
				<div class="border-border/60 border-t bg-background md:hidden">
					<div class="mx-auto flex max-w-[1400px] flex-col gap-3 px-4 py-4 sm:px-6">
						<DateRangePicker />
						<div class="flex items-center justify-between">
							<ThemeToggle />
							<Button variant="ghost" size="sm" onclick={logout}>Logout</Button>
						</div>
					</div>
				</div>
			{/if}
		</header>
	{/if}

	<main class="mx-auto max-w-[1400px] px-4 py-6 sm:px-6">
		{@render children?.()}
	</main>
</div>
