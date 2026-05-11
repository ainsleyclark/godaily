/**
 * app.ts
 *
 * @author Ainsley Clark
 * @author URL:   https://ainsley.dev
 * @author Email: hello@ainsley.dev
 */

import { initSubscribeForm } from './subscribe';
import { initSwipers } from './swiper';
import { initLogoTicker } from './logos';
import { initHamburger } from './hamburger';

document.addEventListener('DOMContentLoaded', () => {
	initSubscribeForm();
	initSwipers();
	initLogoTicker();
	initHamburger();
});
