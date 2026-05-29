/**
 * drawer.ts
 *
 * Wiring for the reusable off-canvas Drawer component (see
 * views/components/drawer.templ). Listeners are delegated from the document so
 * they survive htmx swaps — the browse filter drawer's trigger and close
 * controls live inside regions that htmx replaces on every filter change.
 *
 * Open state is held as `is-open` on the `[data-drawer]` wrapper, which sits
 * outside the swapped panel, so the drawer stays open across those swaps.
 *
 * @author Ainsley Clark
 * @author URL:   https://ainsley.dev
 * @author Email: hello@ainsley.dev
 */

const OPEN_CLASS = 'is-open';
const BODY_LOCK = 'drawer-open';

const syncTriggers = (id: string, open: boolean): void => {
	document
		.querySelectorAll<HTMLElement>(`[data-drawer-open="${id}"]`)
		.forEach((el) => el.setAttribute('aria-expanded', String(open)));
};

const open = (drawer: HTMLElement): void => {
	drawer.classList.add(OPEN_CLASS);
	document.body.classList.add(BODY_LOCK);
	syncTriggers(drawer.id, true);
};

const close = (drawer: HTMLElement): void => {
	drawer.classList.remove(OPEN_CLASS);
	if (!document.querySelector(`[data-drawer].${OPEN_CLASS}`)) {
		document.body.classList.remove(BODY_LOCK);
	}
	syncTriggers(drawer.id, false);
};

export const initDrawers = (): void => {
	document.addEventListener('click', (event) => {
		const target = event.target as HTMLElement | null;
		if (!target) {
			return;
		}

		const opener = target.closest<HTMLElement>('[data-drawer-open]');
		if (opener) {
			const id = opener.getAttribute('data-drawer-open');
			const drawer = id ? document.getElementById(id) : null;
			if (drawer) {
				open(drawer);
			}
			return;
		}

		const closer = target.closest<HTMLElement>('[data-drawer-close]');
		if (closer) {
			const drawer = closer.closest<HTMLElement>('[data-drawer]');
			if (drawer) {
				close(drawer);
			}
		}
	});

	document.addEventListener('keydown', (event) => {
		if (event.key !== 'Escape') {
			return;
		}
		document
			.querySelectorAll<HTMLElement>(`[data-drawer].${OPEN_CLASS}`)
			.forEach(close);
	});
};
