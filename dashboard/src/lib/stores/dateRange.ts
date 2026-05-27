import { writable } from 'svelte/store';
import type { Bucket } from '$lib/api/types';

export type RangePreset = '7d' | '30d' | '90d' | 'ytd' | 'custom';

export interface DateRangeState {
	from: Date;
	to: Date;
	bucket: Bucket;
	preset: RangePreset;
}

function startOfDay(d: Date): Date {
	const x = new Date(d);
	x.setHours(0, 0, 0, 0);
	return x;
}

function endOfDay(d: Date): Date {
	const x = new Date(d);
	x.setHours(23, 59, 59, 999);
	return x;
}

function daysAgo(n: number): Date {
	const d = new Date();
	d.setDate(d.getDate() - n);
	return startOfDay(d);
}

export function presetRange(preset: RangePreset, current?: DateRangeState): DateRangeState {
	const to = endOfDay(new Date());
	switch (preset) {
		case '7d':
			return { from: daysAgo(7), to, bucket: 'day', preset };
		case '30d':
			return { from: daysAgo(30), to, bucket: 'day', preset };
		case '90d':
			return { from: daysAgo(90), to, bucket: 'week', preset };
		case 'ytd': {
			const from = new Date(new Date().getFullYear(), 0, 1);
			return { from, to, bucket: 'week', preset };
		}
		case 'custom':
			return current ?? presetRange('30d');
	}
}

export const dateRange = writable<DateRangeState>(presetRange('30d'));

export function toQueryParams(r: DateRangeState): { from: string; to: string; bucket: Bucket } {
	return {
		from: r.from.toISOString().slice(0, 10),
		to: r.to.toISOString().slice(0, 10),
		bucket: r.bucket
	};
}
