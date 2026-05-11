import InfiniteMarquee from 'vanilla-infinite-marquee';

export function initLogoTicker(): void {
	const el = document.querySelector<HTMLElement>('.logo-ticker');
	if (!el) return;
	new InfiniteMarquee({
		element: el,
		direction: 'left',
		speed: 50000,
		spaceBetween: '4rem',
		pauseOnHover: false,
	});
}
