import Swiper from 'swiper';

export function initSwipers(): void {
	const instances: Swiper[] = [];

	document.querySelectorAll<HTMLElement>('.section-swiper').forEach(el => {
		instances.push(new Swiper(el, {
			slidesPerView: 1.15,
			spaceBetween: 16,
			observer: true,
			observeParents: true,
			breakpoints: {
				900: { enabled: false },
			},
		}));
	});

	window.addEventListener('orientationchange', () => {
		setTimeout(() => instances.forEach(sw => sw.update()), 300);
	});
}
