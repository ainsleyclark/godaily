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

	function logout() {
		auth.clearSecret();
		void goto('/login');
	}
</script>

<Toaster theme="dark" position="top-right" richColors />

<div class="min-h-screen">
	{#if !isLogin}
		<header
			class="border-border/60 bg-background/80 sticky top-0 z-40 border-b backdrop-blur"
		>
			<div class="mx-auto flex h-14 max-w-[1400px] items-center gap-4 px-6">
				<div class="flex items-center gap-2">
					<span
						class="inline-flex h-6 w-6 items-center justify-center rounded-md text-xs font-bold"
						style="background:var(--chart-1); color:var(--primary-foreground)"
					>
						G
					</span>
					<span class="text-sm font-semibold tracking-tight">GoDaily · Mission Control</span>
				</div>
				<div class="ml-auto flex items-center gap-2">
					<DateRangePicker />
					<div class="bg-border mx-1 h-6 w-px"></div>
					<ThemeToggle />
					<Button variant="ghost" size="sm" onclick={logout}>Logout</Button>
				</div>
			</div>
		</header>
	{/if}

	<main class="mx-auto max-w-[1400px] px-6 py-6">
		{@render children?.()}
	</main>
</div>
