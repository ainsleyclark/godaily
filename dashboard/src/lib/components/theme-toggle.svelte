<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import { Sun, Moon } from '@lucide/svelte';

	let isDark = $state(true);

	$effect(() => {
		if (typeof document === 'undefined') return;
		const stored = localStorage.getItem('godaily_theme');
		isDark = stored ? stored === 'dark' : true;
		document.documentElement.classList.toggle('dark', isDark);
	});

	function toggle() {
		isDark = !isDark;
		document.documentElement.classList.toggle('dark', isDark);
		localStorage.setItem('godaily_theme', isDark ? 'dark' : 'light');
	}
</script>

<Button variant="ghost" size="icon" onclick={toggle} aria-label="Toggle theme">
	{#if isDark}
		<Sun class="h-4 w-4" />
	{:else}
		<Moon class="h-4 w-4" />
	{/if}
</Button>
