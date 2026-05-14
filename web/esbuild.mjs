/**
 * This script sets up a development environment using esbuild and BrowserSync.
 * It supports two modes:
 * 1. Build Mode: Compiles the source files once.
 * 2. Watch Mode: Compiles the source files and sets up file watchers to automatically
 *    rebuild and reload the browser on file changes.
 *
 * Usage:
 * - Build Mode: `node esbuild.js`
 * - Watch Mode: `node esbuild.js --watch`
 */
import * as esbuild from 'esbuild';
import { copy } from 'esbuild-plugin-copy';
import { sassPlugin } from 'esbuild-sass-plugin';
import copyAndConvertImages from './bin/images.mjs';
import { svgoPlugin } from './bin/svgo.mjs';

const isProd = !process.argv.includes('--watch');
const excludeImages = process.argv.includes('--exclude-images');

/**
 * ESBuild options for building the project.
 *
 * @type {import('esbuild').CommonOptions}
 */
const options = {
	entryPoints: [
		{ in: 'assets/js/app.ts', out: 'js/app' },
		{ in: 'assets/scss/app.scss', out: 'css/app' },
	],
	bundle: true,
	outdir: 'dist',
	logLevel: 'info',
	sourcemap: isProd ? false : 'inline',
	loader: {
		'.woff': 'file',
		'.woff2': 'file',
		'.ttf': 'file',
		'.svg': 'file',
		'.gif': 'file',
		'.jpg': 'file',
		'.png': 'file'
	},
	external: [
		'*.woff',
		'*.woff2',
		'*.ttf',
	],
	plugins: [
		sassPlugin({
			loadPaths: ['./node_modules'],
			logger: {
				warn: (message, options) => {
					if (options?.span?.url && String(options.span.url).includes('node_modules')) {
						return;
					}
					console.warn(message);
				}
			}
		}),
		copy({
			assets: [
				{
					from: ['./assets/fonts/**/*'],
					to: ['./fonts'],
				},
				{
					from: ['./assets/favicon/**/*'],
					to: ['./favicon'],
				},
				{
					from: ['./assets/blobs/**/*'],
					to: ['./blobs'],
				},
				{
					from: ['./assets/favicon.ico'],
					to: ['./'],
				},
				{
					from: ['./assets/images/**/*.svg'],
					to: ['./images'],
				},
				{
					from: ['./assets/images/logos/*.png'],
					to: ['./images/logos'],
				},
			],
		}),
		svgoPlugin(),
	],
	minify: isProd,
	allowOverwrite: true,
};

(async () => {
	// Check for watch flag in arguments
	if (isProd) {
		await esbuild.build(options);
		if (!excludeImages) {
			await copyAndConvertImages('assets/images', 'dist/images');
			await copyAndConvertImages('assets/favicon', 'dist/favicon');
		}
	} else {
		try {
			if (!excludeImages) {
				await copyAndConvertImages('assets/images', 'dist/images');
				await copyAndConvertImages('assets/favicon', 'dist/favicon');
			}
			const ctx = await esbuild.context(options);
			await ctx.watch();
			await ctx.serve({
				port: 3002,
				host: 'localhost',
			});
			console.log('👀 Watching for changes (live-reload SSE on :3002/esbuild)...');
		} catch (err) {
			console.error('Watch failed:', err);
		}
	}
})();
