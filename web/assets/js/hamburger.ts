export function initHamburger(): void {
	const btn = document.querySelector<HTMLButtonElement>('.header__hamburger');
	const header = document.querySelector<HTMLElement>('.header');
	const nav = document.getElementById('primary-nav');

	if (!btn || !header || !nav) return;

	const close = () => {
		header.classList.remove('is-active');
		btn.classList.remove('is-active');
		btn.setAttribute('aria-expanded', 'false');
	};

	btn.addEventListener('click', () => {
		const isActive = header.classList.toggle('is-active');
		btn.classList.toggle('is-active', isActive);
		btn.setAttribute('aria-expanded', String(isActive));
	});

	nav.querySelectorAll<HTMLAnchorElement>('.nav__link').forEach((link) => {
		link.addEventListener('click', close);
	});

	document.addEventListener('keydown', (e) => {
		if (e.key === 'Escape' && header.classList.contains('is-active')) {
			close();
			btn.focus();
		}
	});

	document.addEventListener('click', (e) => {
		if (!header.contains(e.target as Node)) close();
	});
}
