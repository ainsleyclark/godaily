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
		opens: 'Opens',
		clicks: 'Clicks',
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

	const W = 600;
	const H = 220;
	const PAD = { top: 16, right: 16, bottom: 28, left: 44 };

	function x(i: number, n: number) {
		if (n <= 1) return PAD.left;
		return PAD.left + (i / (n - 1)) * (W - PAD.left - PAD.right);
	}
	function y(v: number) {
		const range = maxValue - minValue || 1;
		return H - PAD.bottom - ((v - minValue) / range) * (H - PAD.top - PAD.bottom);
	}

	const linePath = $derived.by(() => {
		if (!series.length) return '';
		return series.map((p, i) => `${i === 0 ? 'M' : 'L'} ${x(i, series.length)} ${y(p.value)}`).join(' ');
	});

	const areaPath = $derived.by(() => {
		if (!series.length) return '';
		const top = series
			.map((p, i) => `${i === 0 ? 'M' : 'L'} ${x(i, series.length)} ${y(p.value)}`)
			.join(' ');
		const last = x(series.length - 1, series.length);
		const first = x(0, series.length);
		return `${top} L ${last} ${H - PAD.bottom} L ${first} ${H - PAD.bottom} Z`;
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

	function fmt(v: number): string {
		return isRate ? formatPercent(v) : formatCompact(v);
	}

	let hover = $state<number | null>(null);
</script>

<Card>
	<CardHeader>
		<div class="flex items-start justify-between">
			<div>
				<CardTitle>Engagement trend</CardTitle>
				<CardDescription>
					{#if data}
						{metricLabels[metric]} · {data.bucket}
					{:else}
						Loading…
					{/if}
				</CardDescription>
			</div>
			<div class="bg-secondary/40 flex items-center rounded-md p-0.5">
				{#each ['opens', 'clicks', 'open_rate', 'click_rate'] as TrendMetric[] as m}
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
			<Skeleton class="h-[220px] w-full" />
		{:else if !series.length}
			<div class="text-muted-foreground flex h-[220px] items-center justify-center text-sm">
				No data
			</div>
		{:else}
			<svg viewBox="0 0 {W} {H}" class="h-[220px] w-full" preserveAspectRatio="none"
				onmouseleave={() => (hover = null)}>
				<defs>
					<linearGradient id="trend-grad" x1="0" x2="0" y1="0" y2="1">
						<stop offset="0%" stop-color="var(--chart-1)" stop-opacity="0.35" />
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
				<path d={linePath} fill="none" stroke="var(--chart-1)" stroke-width="2" />
				{#each series as p, i (i)}
					<circle
						cx={x(i, series.length)}
						cy={y(p.value)}
						r={hover === i ? 4 : 0}
						fill="var(--chart-1)"
						stroke="var(--background)"
						stroke-width="2"
					/>
				{/each}
				<!-- hover layer -->
				{#each series as _, i (i)}
					<rect
						x={x(i, series.length) - (W / series.length) / 2}
						y={PAD.top}
						width={W / series.length}
						height={H - PAD.top - PAD.bottom}
						fill="transparent"
						onmouseenter={() => (hover = i)}
					/>
				{/each}
				{#if hover !== null}
					<line
						x1={x(hover, series.length)}
						x2={x(hover, series.length)}
						y1={PAD.top}
						y2={H - PAD.bottom}
						stroke="var(--muted-foreground)"
						stroke-width="0.5"
					/>
				{/if}
			</svg>
			{#if hover !== null}
				<div class="text-muted-foreground mt-2 flex justify-between text-xs">
					<span>{formatDateShort(series[hover].date.toISOString())}</span>
					<span class="text-foreground font-medium tabular-nums">{fmt(series[hover].value)}</span>
				</div>
			{/if}
		{/if}
	</CardContent>
</Card>
