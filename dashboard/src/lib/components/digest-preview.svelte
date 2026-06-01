<script lang="ts">
	import type { DigestItem } from '$lib/api/types';
	import { groupBySection } from '$lib/digest/sections';
	import { Badge } from '$lib/components/ui/badge';
	import { ExternalLink } from '@lucide/svelte';

	interface Props {
		items: DigestItem[];
	}
	let { items }: Props = $props();

	const sections = $derived(groupBySection(items ?? []));
</script>

{#if sections.length === 0}
	<div class="text-muted-foreground p-8 text-center text-sm">No items in this issue</div>
{:else}
	<div class="space-y-8">
		{#each sections as section (section.tag)}
			<section>
				<header class="mb-3 flex items-center gap-2 border-b pb-2">
					<h3 class="text-base font-semibold tracking-tight">{section.title}</h3>
					<Badge variant="secondary">{section.items.length}</Badge>
				</header>
				<ul class="space-y-4">
					{#each section.items as item (item.id)}
						<li class="space-y-1">
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
							<div class="text-muted-foreground flex items-center gap-2 text-xs">
								<span class="capitalize">{item.source}</span>
								{#if item.author?.name || item.author?.username}
									<span>·</span>
									<span>{item.author.name || item.author.username}</span>
								{/if}
							</div>
							{#if item.snippet}
								<p class="text-muted-foreground text-sm leading-relaxed">{item.snippet}</p>
							{/if}
						</li>
					{/each}
				</ul>
			</section>
		{/each}
	</div>
{/if}
