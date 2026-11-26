// Platform-aware post length, mirroring pkg/services/social/compose.
// Mastodon counts every URL as 23 characters, and Bluesky shows URLs
// shortened (the link facet keeps the full URL), so a raw .length
// overstates link-heavy posts like the weekly recap.

const URL_PATTERN = /https?:\/\/\S+/g;
const TRAILING_PUNCT = /[.,;:!?)'"]+$/;
const MASTODON_URL_LENGTH = 23;
const BLUESKY_DISPLAY_MAX = 30;

const runes = (s: string): number => Array.from(s).length;

// blueskyDisplay mirrors compose.ShortenURL: no scheme, no "www.", no
// utm_* parameters and no trailing slash, capped at 30 runes.
function blueskyDisplay(raw: string): string {
	let display = raw;
	try {
		const u = new URL(raw);
		for (const key of [...u.searchParams.keys()]) {
			if (key.startsWith('utm_')) u.searchParams.delete(key);
		}
		display = u.host.replace(/^www\./, '') + u.pathname;
		display = display.replace(/\/$/, '');
		const q = u.searchParams.toString();
		if (q) display += '?' + q;
	} catch {
		// Not a parseable URL: count it as-is, capped below.
	}
	const chars = Array.from(display);
	return chars.length <= BLUESKY_DISPLAY_MAX
		? display
		: chars.slice(0, BLUESKY_DISPLAY_MAX - 1).join('') + '…';
}

function urlLength(platform: string, url: string): number {
	switch (platform) {
		case 'mastodon':
			return MASTODON_URL_LENGTH;
		case 'bluesky':
			return runes(blueskyDisplay(url));
		default:
			return runes(url);
	}
}

// postLength reports how many characters text occupies on platform.
export function postLength(platform: string, text: string): number {
	const p = platform.toLowerCase();
	if (p !== 'mastodon' && p !== 'bluesky') return runes(text);

	let n = 0;
	let last = 0;
	for (const m of text.matchAll(URL_PATTERN)) {
		const raw = m[0];
		const start = m.index ?? 0;
		const url = raw.replace(TRAILING_PUNCT, '');
		n += runes(text.slice(last, start));
		n += urlLength(p, url);
		n += runes(raw.slice(url.length));
		last = start + raw.length;
	}
	return n + runes(text.slice(last));
}
