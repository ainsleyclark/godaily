import sharp from 'sharp';
import { readdir, mkdir } from 'fs/promises';
import { join, extname, basename } from 'path';

const RASTER_EXTS = new Set(['.jpg', '.jpeg', '.png', '.webp']);
const MAX_WIDTH = 920; // 2× the 460 px CSS max-width

async function processDir(src, dest) {
	await mkdir(dest, { recursive: true });
	const entries = await readdir(src, { withFileTypes: true });
	for (const entry of entries) {
		const srcPath = join(src, entry.name);
		if (entry.isDirectory()) {
			await processDir(srcPath, join(dest, entry.name));
			continue;
		}
		const ext = extname(entry.name).toLowerCase();
		if (!RASTER_EXTS.has(ext)) continue;
		const name = basename(entry.name, ext);
		const outPath = join(dest, `${name}.png`);
		await sharp(srcPath)
			.resize({ width: MAX_WIDTH, withoutEnlargement: true })
			.png({ compressionLevel: 9 })
			.toFile(outPath);
		console.log(`  image: ${outPath}`);
	}
}

export default async function copyAndConvertImages(src, dest) {
	await processDir(src, dest);
}
