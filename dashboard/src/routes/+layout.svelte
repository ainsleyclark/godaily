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
	const currentPath = $derived($page.url.pathname);

	let mobileOpen = $state(false);

	function logout() {
		auth.clearSecret();
		void goto('/login');
	}

	const navItems = [
		{
			href: '/',
			label: 'Overview',
			icon: `<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/></svg>`
		},
		{
			href: '/issues',
			label: 'Issues',
			icon: `<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/><polyline points="10 9 9 9 8 9"/></svg>`
		},
		{
			href: '/subscribers',
			label: 'Subscribers',
			icon: `<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>`
		},
		{
			href: '/content',
			label: 'Content',
			icon: `<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="8" y1="6" x2="21" y2="6"/><line x1="8" y1="12" x2="21" y2="12"/><line x1="8" y1="18" x2="21" y2="18"/><line x1="3" y1="6" x2="3.01" y2="6"/><line x1="3" y1="12" x2="3.01" y2="12"/><line x1="3" y1="18" x2="3.01" y2="18"/></svg>`
		},
		{
			href: '/social',
			label: 'Social',
			icon: `<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 2h-3a5 5 0 0 0-5 5v3H7v4h3v8h4v-8h3l1-4h-4V7a1 1 0 0 1 1-1h3z"/></svg>`
		}
	];

	function isActive(href: string) {
		if (href === '/') return currentPath === '/';
		return currentPath.startsWith(href);
	}
</script>

<Toaster theme="dark" position="top-right" richColors />

{#if isLogin}
	<main class="mx-auto max-w-[1400px] px-4 py-6 sm:px-6">
		{@render children?.()}
	</main>
{:else}
	<div class="flex min-h-screen">
		<!-- Sidebar -->
		<aside class="border-border/60 bg-background hidden w-52 shrink-0 flex-col border-r lg:flex">
			<div class="flex h-14 items-center gap-2 px-4 border-b border-border/60">
				<img src="/favicon.png" alt="GoDaily" class="h-6 w-6 rounded-md" />
				<span class="text-sm font-semibold tracking-tight">GoDaily</span>
			</div>

			<nav class="flex flex-col gap-1 p-3 flex-1">
				{#each navItems as item (item.href)}
					<a
						href={item.href}
						class="flex items-center gap-2.5 rounded-md px-3 py-2 text-sm transition-colors
							{isActive(item.href)
							? 'bg-primary text-primary-foreground font-medium'
							: 'text-muted-foreground hover:bg-accent hover:text-foreground'}"
					>
						{@html item.icon}
						{item.label}
					</a>
				{/each}
			</nav>

			<div class="border-t border-border/60 p-3 flex items-center justify-between">
				<ThemeToggle />
				<Button variant="ghost" size="sm" onclick={logout} class="text-muted-foreground text-xs">
					Logout
				</Button>
			</div>
		</aside>

		<!-- Main content -->
		<div class="flex min-w-0 flex-1 flex-col">
			<!-- Top bar -->
			<header class="border-border/60 bg-background/80 sticky top-0 z-40 border-b backdrop-blur">
				<div class="flex h-14 items-center gap-4 px-4 sm:px-6">
					<!-- Mobile brand + menu toggle -->
					<div class="flex items-center gap-2 lg:hidden">
						<img src="/favicon.png" alt="GoDaily" class="h-6 w-6 rounded-md" />
						<span class="text-sm font-semibold tracking-tight">GoDaily</span>
					</div>

					<div class="ml-auto hidden items-center gap-2 md:flex">
						<DateRangePicker />
					</div>

					<!-- Mobile toggle -->
					<button
						type="button"
						class="ml-auto inline-flex h-9 w-9 items-center justify-center rounded-md lg:hidden hover:bg-accent"
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
					<div class="border-border/60 border-t bg-background lg:hidden">
						<nav class="flex flex-col gap-1 p-3">
							{#each navItems as item (item.href)}
								<a
									href={item.href}
									onclick={() => (mobileOpen = false)}
									class="flex items-center gap-2.5 rounded-md px-3 py-2 text-sm transition-colors
										{isActive(item.href)
										? 'bg-primary text-primary-foreground font-medium'
										: 'text-muted-foreground hover:bg-accent hover:text-foreground'}"
								>
									{@html item.icon}
									{item.label}
								</a>
							{/each}
						</nav>
						<div class="border-t border-border/60 px-4 py-3 flex items-center gap-3">
							<DateRangePicker />
							<ThemeToggle />
							<Button variant="ghost" size="sm" onclick={logout}>Logout</Button>
						</div>
					</div>
				{/if}
			</header>

			<main class="flex-1 px-4 py-6 sm:px-6">
				{@render children?.()}
			</main>
		</div>
	</div>
{/if}
