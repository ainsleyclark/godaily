import Swiper from 'swiper';
import 'swiper/css';

export function initSwipers(): void {
	document.querySelectorAll<HTMLElement>('.section-swiper').forEach(el => {
		new Swiper(el, {
			slidesPerView: 1.15,
			spaceBetween: 16,
			breakpoints: {
				900: { enabled: false },
			},
		});
	});
}
