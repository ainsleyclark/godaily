const nfCompact = new Intl.NumberFormat('en-US', { notation: 'compact', maximumFractionDigits: 1 });
const nfPlain = new Intl.NumberFormat('en-US');

export function formatNumber(n: number | null | undefined): string {
	if (n == null) return '--';
	return nfPlain.format(n);
}

export function formatCompact(n: number | null | undefined): string {
	if (n == null) return '--';
	return nfCompact.format(n);
}

export function formatPercent(n: number | null | undefined, digits = 1): string {
	if (n == null) return '--';
	// API returns rates as 0..1 typically; if > 1, assume already a percent.
	const v = n <= 1 ? n * 100 : n;
	return `${v.toFixed(digits)}%`;
}

export function formatDate(iso: string | null | undefined): string {
	if (!iso) return '--';
	const d = new Date(iso);
	if (Number.isNaN(d.getTime())) return iso;
	return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
}

export function formatDateShort(iso: string | null | undefined): string {
	if (!iso) return '--';
	const d = new Date(iso);
	if (Number.isNaN(d.getTime())) return iso;
	return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
}

// formatTimeShort renders the hour of day (e.g. "14:00"), for intraday/hourly
// series where the date alone would repeat across every bucket.
export function formatTimeShort(iso: string | null | undefined): string {
	if (!iso) return '--';
	const d = new Date(iso);
	if (Number.isNaN(d.getTime())) return iso;
	return d.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', hour12: false });
}

// formatDateTimeShort renders date and hour together (e.g. "Jun 5, 14:00") for
// hourly tooltips where both the day and the hour matter.
export function formatDateTimeShort(iso: string | null | undefined): string {
	if (!iso) return '--';
	const d = new Date(iso);
	if (Number.isNaN(d.getTime())) return iso;
	return d.toLocaleString('en-US', {
		month: 'short',
		day: 'numeric',
		hour: '2-digit',
		minute: '2-digit',
		hour12: false
	});
}
