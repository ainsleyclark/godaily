<script lang="ts">
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import type { TrendData, TrendMetric } from '$lib/api/types';
	import { formatDateShort, formatCompact, formatPercent } from '$lib/utils/format';

	interface Props {
		data: TrendData | null;
		loading?: boolean;
		metric: TrendMetric;
		onMetricChange: (m: TrendMetric) => void;
	}
	let { data, loading = false, metric, onMetricChange }: Props = $props();

	const metricLabels: Record<TrendMetric, string> = {
		unique_opens: 'Opens',
		unique_clicks: 'Clicks',
		open_rate: 'Open rate',
		click_rate: 'Click rate'
	};

	const isRate = $derived(metric === 'open_rate' || metric === 'click_rate');

	const series = $derived(
		(data?.points ?? []).map((p) => ({
			date: new Date(p.bucket_start),
			value: p.value
		}))
	);

	const maxValue = $derived(series.length ? Math.max(...series.map((p) => p.value), 0) : 0);
	const minValue = $derived(series.length ? Math.min(...series.map((p) => p.value), 0) : 0);

	let containerW = $state(600);
	const W = $derived(containerW || 600);
	const H = 240;
	const PAD = { top: 16, right: 16, bottom: 32, left: 48 };

	function x(i: number, n: number) {
		// Centre a lone point rather than pinning it to the left axis, so a
		// just-sent issue (a single day's bucket) still reads as a real datum.
		if (n <= 1) return PAD.left + (W - PAD.left - PAD.right) / 2;
		return PAD.left + (i / (n - 1)) * (W - PAD.left - PAD.right);
	}
	function y(v: number) {
		const range = maxValue - minValue || 1;
		return H - PAD.bottom - ((v - minValue) / range) * (H - PAD.top - PAD.bottom);
	}

	// Monotone-cubic interpolated path (shadcn/recharts-style smooth curve).
	function smoothPath(pts: { px: number; py: number }[]): string {
		if (!pts.length) return '';
		if (pts.length === 1) return `M ${pts[0].px} ${pts[0].py}`;

		const n = pts.length;
		const dx: number[] = [];
		const dy: number[] = [];
		const m: number[] = [];

		for (let i = 0; i < n - 1; i++) {
			dx.push(pts[i + 1].px - pts[i].px);
			dy.push(pts[i + 1].py - pts[i].py);
		}
		const slopes = dy.map((d, i) => d / (dx[i] || 1));
		m.push(slopes[0]);
		for (let i = 1; i < n - 1; i++) {
			if (slopes[i - 1] * slopes[i] <= 0) {
				m.push(0);
				continue;
			}
			m.push((slopes[i - 1] + slopes[i]) / 2);
		}
		m.push(slopes[slopes.length - 1]);

		let d = `M ${pts[0].px} ${pts[0].py}`;
		for (let i = 0; i < n - 1; i++) {
			const h = dx[i];
			const c1x = pts[i].px + h / 3;
			const c1y = pts[i].py + (m[i] * h) / 3;
			const c2x = pts[i + 1].px - h / 3;
			const c2y = pts[i + 1].py - (m[i + 1] * h) / 3;
			d += ` C ${c1x} ${c1y} ${c2x} ${c2y} ${pts[i + 1].px} ${pts[i + 1].py}`;
		}
		return d;
	}

	const projected = $derived(series.map((p, i) => ({ px: x(i, series.length), py: y(p.value) })));
	const linePath = $derived(smoothPath(projected));
	const areaPath = $derived.by(() => {
		if (!projected.length) return '';
		const last = projected[projected.length - 1].px;
		const first = projected[0].px;
		return `${linePath} L ${last} ${H - PAD.bottom} L ${first} ${H - PAD.bottom} Z`;
	});

	const yTicks = $derived.by(() => {
		const n = 4;
		const range = maxValue - minValue || 1;
		return Array.from({ length: n + 1 }, (_, i) => minValue + (range * i) / n);
	});

	const xTicks = $derived.by(() => {
		if (series.length <= 6) return series.map((_, i) => i);
		const step = Math.ceil(series.length / 6);
		return series.map((_, i) => i).filter((i) => i % step === 0);
	});

	// A smooth line needs at least two points to be drawn; render explicit dots
	// for short series so single/sparse windows aren't mistaken for an empty chart.
	const showDots = $derived(series.length > 0 && series.length <= 31);

	function fmt(v: number): string {
		return isRate ? formatPercent(v) : formatCompact(v);
	}

	let hover = $state<number | null>(null);
</script>

<Card>
	<CardHeader>
		<div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
			<div>
				<CardTitle>Engagement trend</CardTitle>
				<CardDescription>
					{#if data}
						{metricLabels[metric]} &middot; {data.bucket}
					{:else}
						Loading…
					{/if}
				</CardDescription>
			</div>
			<div class="bg-secondary/40 flex w-fit max-w-full flex-wrap items-center rounded-md p-0.5">
				{#each ['unique_opens', 'unique_clicks', 'open_rate', 'click_rate'] as TrendMetric[] as m}
					<button
						type="button"
						onclick={() => onMetricChange(m)}
						class="rounded px-2 py-1 text-[11px] font-medium transition-colors"
						class:bg-background={metric === m}
						class:text-foreground={metric === m}
						class:shadow-sm={metric === m}
						class:text-muted-foreground={metric !== m}
					>
						{metricLabels[m]}
					</button>
				{/each}
			</div>
		</div>
	</CardHeader>
	<CardContent>
		{#if loading && !data}
			<Skeleton class="h-[240px] w-full" />
		{:else if !series.length}
			<div class="text-muted-foreground flex h-[240px] items-center justify-center text-sm">
				No data
			</div>
		{:else}
			<div class="w-full" bind:clientWidth={containerW}>
			<svg
				viewBox="0 0 {W} {H}"
				class="h-[240px] w-full"
				role="img"
				aria-label="Engagement trend chart"
				onmouseleave={() => (hover = null)}
			>
				<defs>
					<linearGradient id="trend-grad" x1="0" x2="0" y1="0" y2="1">
						<stop offset="0%" stop-color="var(--chart-1)" stop-opacity="0.5" />
						<stop offset="100%" stop-color="var(--chart-1)" stop-opacity="0" />
					</linearGradient>
				</defs>
				{#each yTicks as t, i (i)}
					<line
						x1={PAD.left}
						x2={W - PAD.right}
						y1={y(t)}
						y2={y(t)}
						stroke="var(--border)"
						stroke-dasharray="2 4"
						stroke-width="0.5"
					/>
					<text
						x={PAD.left - 8}
						y={y(t)}
						dy="0.32em"
						text-anchor="end"
						class="fill-muted-foreground text-[10px]"
					>
						{fmt(t)}
					</text>
				{/each}
				{#each xTicks as i (i)}
					<text
						x={x(i, series.length)}
						y={H - PAD.bottom + 16}
						text-anchor="middle"
						class="fill-muted-foreground text-[10px]"
					>
						{formatDateShort(series[i].date.toISOString())}
					</text>
				{/each}
				<path d={areaPath} fill="url(#trend-grad)" />
				<path d={linePath} fill="none" stroke="var(--chart-1)" stroke-width="2" stroke-linejoin="round" stroke-linecap="round" />
				{#if showDots}
					{#each series as p, i (i)}
						<circle cx={x(i, series.length)} cy={y(p.value)} r="2.5" fill="var(--chart-1)" />
					{/each}
				{/if}
				{#if hover !== null}
					<line
						x1={x(hover, series.length)}
						x2={x(hover, series.length)}
						y1={PAD.top}
						y2={H - PAD.bottom}
						stroke="var(--muted-foreground)"
						stroke-width="0.5"
					/>
					<circle
						cx={x(hover, series.length)}
						cy={y(series[hover].value)}
						r="4"
						fill="var(--chart-1)"
						stroke="var(--background)"
						stroke-width="2"
					/>
				{/if}
				{#each series as _, i (i)}
					<rect
						x={x(i, series.length) - (W / series.length) / 2}
						y={PAD.top}
						width={W / series.length}
						height={H - PAD.top - PAD.bottom}
						fill="transparent"
						role="presentation"
						onmouseenter={() => (hover = i)}
					/>
				{/each}
			</svg>
			</div>
			{#if hover !== null}
				<div class="text-muted-foreground mt-2 flex justify-between text-xs">
					<span>{formatDateShort(series[hover].date.toISOString())}</span>
					<span class="text-foreground font-medium tabular-nums">{fmt(series[hover].value)}</span>
				</div>
			{/if}
		{/if}
	</CardContent>
</Card>
