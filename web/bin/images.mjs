import sharp from 'sharp';
import { readdir, mkdir } from 'fs/promises';
import { join, extname, basename } from 'path';

const RASTER_EXTS = new Set(['.jpg', '.jpeg', '.png', '.webp']);

// Per-directory size caps (width × height, null = unconstrained).
// Values are 2× the CSS display size for retina sharpness.
const DIR_OPTS = {
	marks: { maxWidth: 64,  maxHeight: 64  },
	logos: { maxWidth: 214, maxHeight: null },
};
const DEFAULT_OPTS = { maxWidth: 920, maxHeight: null };

function optsFor(dirName) {
	return DIR_OPTS[dirName] ?? DEFAULT_OPTS;
}

async function processDir(src, dest, opts) {
	await mkdir(dest, { recursive: true });
	const entries = await readdir(src, { withFileTypes: true });
	for (const entry of entries) {
		const srcPath = join(src, entry.name);
		if (entry.isDirectory()) {
			await processDir(srcPath, join(dest, entry.name), optsFor(entry.name));
			continue;
		}
		const ext = extname(entry.name).toLowerCase();
		if (!RASTER_EXTS.has(ext)) continue;
		const name = basename(entry.name, ext);

		const resize = {
			width: opts.maxWidth ?? undefined,
			height: opts.maxHeight ?? undefined,
			withoutEnlargement: true,
			fit: 'inside',
		};

		const img = sharp(srcPath).resize(resize);

		await img.clone().png({ compressionLevel: 9 }).toFile(join(dest, `${name}.png`));
		await img.clone().webp({ quality: 80 }).toFile(join(dest, `${name}.webp`));
		await img.clone().avif({ quality: 60 }).toFile(join(dest, `${name}.avif`));

		console.log(`  image: ${dest}/${name}.{png,webp,avif}`);
	}
}

export default async function copyAndConvertImages(src, dest) {
	await processDir(src, dest, DEFAULT_OPTS);
}
