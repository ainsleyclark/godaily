<script lang="ts">
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import type { SubscriberData } from '$lib/api/types';
	import { formatDateShort, formatCompact } from '$lib/utils/format';

	interface Props {
		data: SubscriberData | null;
		loading?: boolean;
	}
	let { data, loading = false }: Props = $props();

	const points = $derived(data?.points ?? []);

	const W = 600;
	const H = 220;
	const PAD = { top: 16, right: 16, bottom: 28, left: 44 };

	const bars = $derived.by(() => {
		if (!points.length) return [];
		const inner = W - PAD.left - PAD.right;
		const step = inner / points.length;
		const barW = Math.max(2, step * 0.7);
		const maxPos = Math.max(0, ...points.map((p) => p.new + p.confirmed));
		const maxNeg = Math.max(0, ...points.map((p) => p.unsubscribed + p.lost));
		const maxActive = Math.max(1, ...points.map((p) => p.active_at_end));
		return points.map((p, i) => {
			const cx = PAD.left + step * i + step / 2;
			return {
				p,
				i,
				cx,
				barW,
				maxPos,
				maxNeg,
				maxActive
			};
		});
	});

	const maxPos = $derived(bars.length ? bars[0].maxPos : 0);
	const maxNeg = $derived(bars.length ? bars[0].maxNeg : 0);
	const maxActive = $derived(bars.length ? bars[0].maxActive : 1);

	const midY = $derived(PAD.top + ((H - PAD.top - PAD.bottom) * maxPos) / (maxPos + maxNeg || 1));

	function yPos(v: number): number {
		const height = (v / (maxPos || 1)) * (midY - PAD.top);
		return midY - height;
	}
	function yNeg(v: number): number {
		const height = (v / (maxNeg || 1)) * (H - PAD.bottom - midY);
		return midY + height;
	}
	function yActive(v: number): number {
		return PAD.top + (1 - v / maxActive) * (H - PAD.top - PAD.bottom);
	}

	const activeLine = $derived.by(() => {
		if (!points.length) return '';
		const inner = W - PAD.left - PAD.right;
		const step = inner / points.length;
		return points
			.map((p, i) => {
				const cx = PAD.left + step * i + step / 2;
				return `${i === 0 ? 'M' : 'L'} ${cx} ${yActive(p.active_at_end)}`;
			})
			.join(' ');
	});

	let hover = $state<number | null>(null);
</script>

<Card>
	<CardHeader>
		<CardTitle>Subscriber growth</CardTitle>
		<CardDescription>
			{#if data}
				New vs. unsubscribed · active line · {data.bucket}
			{:else}
				Loading…
			{/if}
		</CardDescription>
	</CardHeader>
	<CardContent>
		{#if loading && !points.length}
			<Skeleton class="h-[220px] w-full" />
		{:else if !points.length}
			<div class="text-muted-foreground flex h-[220px] items-center justify-center text-sm">
				No data
			</div>
		{:else}
			<svg viewBox="0 0 {W} {H}" class="h-[220px] w-full" preserveAspectRatio="none"
				onmouseleave={() => (hover = null)}>
				<!-- zero line -->
				<line
					x1={PAD.left}
					x2={W - PAD.right}
					y1={midY}
					y2={midY}
					stroke="var(--border)"
					stroke-width="0.5"
				/>
				{#each bars as b (b.i)}
					<!-- new (positive) -->
					<rect
						x={b.cx - b.barW / 2}
						y={yPos(b.p.new)}
						width={b.barW}
						height={midY - yPos(b.p.new)}
						fill="var(--chart-2)"
						opacity="0.85"
					/>
					<!-- unsubscribed (negative) -->
					<rect
						x={b.cx - b.barW / 2}
						y={midY}
						width={b.barW}
						height={yNeg(b.p.unsubscribed) - midY}
						fill="var(--destructive)"
						opacity="0.7"
					/>
				{/each}
				<!-- active_at_end line -->
				<path d={activeLine} fill="none" stroke="var(--chart-1)" stroke-width="2" />
				<!-- x labels -->
				{#each points as p, i (i)}
					{#if i % Math.max(1, Math.ceil(points.length / 6)) === 0}
						<text
							x={PAD.left + ((W - PAD.left - PAD.right) / points.length) * i + (W - PAD.left - PAD.right) / points.length / 2}
							y={H - PAD.bottom + 16}
							text-anchor="middle"
							class="fill-muted-foreground text-[10px]"
						>
							{formatDateShort(p.bucket_start)}
						</text>
					{/if}
				{/each}
				<!-- hover layer -->
				{#each bars as b (b.i)}
					<rect
						x={b.cx - (W - PAD.left - PAD.right) / points.length / 2}
						y={PAD.top}
						width={(W - PAD.left - PAD.right) / points.length}
						height={H - PAD.top - PAD.bottom}
						fill="transparent"
						onmouseenter={() => (hover = b.i)}
					/>
				{/each}
			</svg>
			<div class="text-muted-foreground mt-2 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs">
				<span class="flex items-center gap-1.5">
					<span class="inline-block h-2 w-2 rounded-sm" style="background:var(--chart-2)"></span>
					New
				</span>
				<span class="flex items-center gap-1.5">
					<span class="inline-block h-2 w-2 rounded-sm" style="background:var(--destructive); opacity:.7"></span>
					Unsubscribed
				</span>
				<span class="flex items-center gap-1.5">
					<span class="inline-block h-0.5 w-3" style="background:var(--chart-1)"></span>
					Active
				</span>
				{#if hover !== null}
					<span class="text-foreground ml-auto tabular-nums">
						{formatDateShort(points[hover].bucket_start)} · +{formatCompact(points[hover].new)} / -{formatCompact(
							points[hover].unsubscribed
						)} · active {formatCompact(points[hover].active_at_end)}
					</span>
				{/if}
			</div>
		{/if}
	</CardContent>
</Card>
