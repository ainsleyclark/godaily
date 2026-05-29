/**
 * app.ts
 *
 * @author Ainsley Clark
 * @author URL:   https://ainsley.dev
 * @author Email: hello@ainsley.dev
 */

import 'htmx.org';
import { initSubscribeForm } from './subscribe';
import { initSwipers } from './swiper';
import { initLogoTicker } from './logos';
import { initHamburger } from './hamburger';
import { initShareButtons } from './share';
import { initBrowse } from './browse';
import { initDrawers } from './drawer';

document.addEventListener('DOMContentLoaded', () => {
	initSubscribeForm();
	initSwipers();
	initLogoTicker();
	initHamburger();
	initShareButtons();
	initBrowse();
	initDrawers();
});
