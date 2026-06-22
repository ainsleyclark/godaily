import type { DigestItem } from '$lib/api/types';

// Mirrors pkg/domain/news/item.go: SectionTags + Tag.Section() + sectionTitles.
export const SECTION_ORDER = [
	'release',
	'proposal_accepted',
	'proposal',
	'conference',
	'discussion',
	'event',
	'article',
	'tutorial',
	'video',
	'trending',
	'security',
	'jobs'
] as const;

export type SectionTag = (typeof SECTION_ORDER)[number];

export const SECTION_TITLES: Record<SectionTag, string> = {
	release: 'Releases',
	proposal_accepted: 'Accepted Proposals',
	proposal: 'Proposals',
	conference: 'Conferences',
	discussion: 'Discussions',
	event: 'Events',
	article: 'Articles',
	tutorial: 'Tutorials',
	video: 'Videos',
	trending: 'Trending',
	security: 'Security',
	jobs: 'Jobs'
};

const FOLD: Record<string, SectionTag> = {
	podcast: 'video',
	proposal_shipped: 'proposal',
	conference_reminder: 'conference',
	conference_alert: 'conference'
};

export function sectionOf(tag: string): SectionTag {
	if (tag in FOLD) return FOLD[tag];
	if ((SECTION_ORDER as readonly string[]).includes(tag)) return tag as SectionTag;
	return 'article';
}

export interface Section {
	tag: SectionTag;
	title: string;
	items: DigestItem[];
}

export function groupBySection(items: DigestItem[]): Section[] {
	const buckets = new Map<SectionTag, DigestItem[]>();
	for (const item of items) {
		const key = sectionOf(item.tag);
		const list = buckets.get(key) ?? [];
		list.push(item);
		buckets.set(key, list);
	}
	const out: Section[] = [];
	for (const tag of SECTION_ORDER) {
		const list = buckets.get(tag);
		if (list && list.length) {
			out.push({ tag, title: SECTION_TITLES[tag], items: list });
		}
	}
	return out;
}
