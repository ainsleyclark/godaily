const overrides: Record<string, string> = {
	dev_to: 'dev.to',
	dev: 'dev.to',
	hacker_news: 'Hacker News',
	hackernews: 'Hacker News',
	hn: 'Hacker News',
	youtube: 'YouTube',
	github: 'GitHub',
	medium: 'Medium',
	reddit: 'Reddit',
	conferences: 'Conferences',
	freecodecamp: 'freeCodeCamp',
	free_code_camp: 'freeCodeCamp',
	stackoverflow: 'Stack Overflow',
	stack_overflow: 'Stack Overflow',
	go_blog: 'Go Blog',
	golang: 'Go',
	go: 'Go',
	x: 'X',
	twitter: 'X'
};

const acronyms = new Set(['ai', 'api', 'cli', 'cpu', 'css', 'db', 'go', 'gpu', 'html', 'http', 'https', 'io', 'json', 'orm', 'os', 'rss', 'sdk', 'sql', 'ssr', 'tcp', 'ui', 'url', 'yaml']);

export function prettify(raw: string | null | undefined): string {
	if (!raw) return '';
	const key = raw.trim().toLowerCase();
	if (overrides[key]) return overrides[key];
	return key
		.replace(/[_-]+/g, ' ')
		.split(/\s+/)
		.filter(Boolean)
		.map((w) => (acronyms.has(w) ? w.toUpperCase() : w[0].toUpperCase() + w.slice(1)))
		.join(' ');
}
