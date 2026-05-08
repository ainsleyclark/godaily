import Swiper from 'swiper';

export function initSwipers(): void {
	document.querySelectorAll<HTMLElement>('.section-swiper').forEach(el => {
		new Swiper(el, {
			slidesPerView: 'auto',
			spaceBetween: 16,
			breakpoints: {
				900: { enabled: false },
			},
		});
	});
}
