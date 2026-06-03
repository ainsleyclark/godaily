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
	import { LayoutGrid, FileText, Users, List, Share2, Menu, X } from '@lucide/svelte';

	let { children } = $props();

	const isLogin = $derived($page.url.pathname === '/login');
	const currentPath = $derived($page.url.pathname);

	// Routes that consume the dateRange store. Sub-routes like /issues/[id]
	// are intentionally excluded — the picker is hidden there.
	const DATE_RANGE_ROUTES = new Set(['/', '/issues', '/subscribers', '/content', '/social']);
	const showDateRange = $derived(DATE_RANGE_ROUTES.has(currentPath));

	let mobileOpen = $state(false);

	function logout() {
		auth.clearSecret();
		void goto('/login');
	}

	const navItems = [
		{ href: '/', label: 'Overview', icon: LayoutGrid },
		{ href: '/issues', label: 'Issues', icon: FileText },
		{ href: '/subscribers', label: 'Subscribers', icon: Users },
		{ href: '/content', label: 'Content', icon: List },
		{ href: '/social', label: 'Social', icon: Share2 }
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
					{@const Icon = item.icon}
					<a
						href={item.href}
						class="flex items-center gap-2.5 rounded-md px-3 py-2 text-sm transition-colors
							{isActive(item.href)
							? 'bg-primary text-primary-foreground font-medium'
							: 'text-muted-foreground hover:bg-accent hover:text-foreground'}"
					>
						<Icon class="h-4 w-4" />
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

					{#if showDateRange}
						<div class="ml-auto hidden items-center gap-2 md:flex">
							<DateRangePicker />
						</div>
					{/if}

					<!-- Mobile toggle -->
					<button
						type="button"
						class="ml-auto inline-flex h-9 w-9 items-center justify-center rounded-md lg:hidden hover:bg-accent"
						aria-label="Toggle menu"
						aria-expanded={mobileOpen}
						onclick={() => (mobileOpen = !mobileOpen)}
					>
						{#if mobileOpen}
							<X class="h-5 w-5" />
						{:else}
							<Menu class="h-5 w-5" />
						{/if}
					</button>
				</div>

				{#if mobileOpen}
					<div class="border-border/60 border-t bg-background lg:hidden">
						<nav class="flex flex-col gap-1 p-3">
							{#each navItems as item (item.href)}
								{@const Icon = item.icon}
								<a
									href={item.href}
									onclick={() => (mobileOpen = false)}
									class="flex items-center gap-2.5 rounded-md px-3 py-2 text-sm transition-colors
										{isActive(item.href)
										? 'bg-primary text-primary-foreground font-medium'
										: 'text-muted-foreground hover:bg-accent hover:text-foreground'}"
								>
									<Icon class="h-4 w-4" />
									{item.label}
								</a>
							{/each}
						</nav>
						<div class="border-t border-border/60 px-4 py-3 flex items-center gap-3">
							{#if showDateRange}
								<DateRangePicker />
							{/if}
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
