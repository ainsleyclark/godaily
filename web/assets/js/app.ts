/**
 * app.ts
 *
 * @author Ainsley Clark
 * @author URL:   https://ainsley.dev
 * @author Email: hello@ainsley.dev
 */

import 'htmx.org';
import { initSubscribeForm } from './subscribe';
import { initLogoTicker } from './logos';
import { initHamburger } from './hamburger';
import { initShareButtons } from './share';
import { initBrowse } from './browse';
import { initDrawers } from './drawer';

document.addEventListener('DOMContentLoaded', () => {
	initSubscribeForm();
	initLogoTicker();
	initHamburger();
	initShareButtons();
	initBrowse();
	initDrawers();
});
