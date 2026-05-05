import { promises as fs } from 'node:fs';
import { resolve } from 'node:path';
import { optimize } from 'svgo';

/**
 * Recursively get all SVG files from a directory.
 *
 * @param {string} dir Directory to read files from
 * @returns {Array<string>} List of file paths
 */
async function getFilesRecursive(dir) {
	const files = [];

	async function scan(dir) {
		const dirents = await fs.readdir(dir, { withFileTypes: true });
		for (const dirent of dirents) {
			const res = resolve(dir, dirent.name);
			if (dirent.isDirectory()) {
				await scan(res);
			} else {
				files.push(res);
			}
		}
	}

	await scan(dir);
	return files.filter((file) => file.endsWith('.svg'));
}

/**
 * SVGO Plugin for esbuild
 */
export function svgoPlugin() {
	return {
		name: 'svgo',
		setup(build) {
			build.onEnd(async () => {
				try {
					const outDir = resolve(build.initialOptions.outdir);
					try { await fs.access(outDir); } catch { return; }
					const files = await getFilesRecursive(outDir);

					for (const file of files) {
						const data = await fs.readFile(file, 'utf8');
						const result = optimize(data, { path: file });
						await fs.writeFile(file, result.data, { flag: 'w' });
					}
				} catch (err) {
					console.error('SVGO Plugin Error:', err);
				}
			});
		},
	};
}
