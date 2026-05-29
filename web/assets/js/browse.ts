/**
 * browse.ts
 *
 * Optimistic active-state for the /browse filter controls. The filter fragment
 * round-trips over the network, so without this a clicked tab or sort segment
 * only lights up once the response swaps in (~1s of dead time). We move the
 * active class on click for instant feedback; htmx then swaps in the
 * authoritative markup, which carries the same state.
 *
 * @author Ainsley Clark
 * @author URL:   https://ainsley.dev
 * @author Email: hello@ainsley.dev
 */

interface ToggleGroup {
	// Selector for an individual clickable item in the group.
	item: string;
	// Selector for the wrapper that scopes the mutually-exclusive items.
	group: string;
	// Class that marks the active item.
	active: string;
}

// Single-select controls where clicking one item deactivates its siblings.
const GROUPS: ToggleGroup[] = [
	{ item: '.tabs__tab', group: '.tabs__scroll', active: 'tabs__tab--active' },
	{ item: '.seg__btn', group: '.seg', active: 'seg__btn--active' },
];

export const initBrowse = (): void => {
	// Delegate from the persistent app root: the tabs and results regions are
	// swapped by htmx, so a listener bound to them wouldn't survive a swap.
	const app = document.querySelector('[data-browse-app]');
	if (!app) {
		return;
	}

	app.addEventListener('click', (event) => {
		const target = event.target as HTMLElement | null;
		if (!target) {
			return;
		}
		for (const { item, group, active } of GROUPS) {
			const clicked = target.closest<HTMLElement>(item);
			if (!clicked) {
				continue;
			}
			const scope = clicked.closest<HTMLElement>(group);
			if (!scope) {
				return;
			}
			scope.querySelectorAll<HTMLElement>(item).forEach((el) => {
				el.classList.remove(active);
				el.removeAttribute('aria-current');
			});
			clicked.classList.add(active);
			clicked.setAttribute('aria-current', 'true');
			return;
		}
	});
};
