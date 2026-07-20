import { chromium } from 'playwright';
import { renderDeckHtml, SCENE_RENDERER_VERSION } from '../build-renderer/render.js';

const EXPECTED_PLAYWRIGHT = '1.60.0';
const readStdin = async () => {
	const chunks = [];
	for await (const chunk of process.stdin) chunks.push(chunk);
	return Buffer.concat(chunks);
};
const browser = await chromium.launch({ headless: true });
try {
	if (process.argv.includes('--ready')) {
		process.stdout.write(
			JSON.stringify({
				ready: true,
				rendererVersion: SCENE_RENDERER_VERSION,
				playwrightVersion: EXPECTED_PLAYWRIGHT,
				chromiumVersion: await browser.version()
			})
		);
		process.exitCode = 0;
	} else {
		const request = JSON.parse((await readStdin()).toString('utf8'));
		const assets = request.assets ?? {};
		const html = renderDeckHtml(request.deck, {
			assetUrl: (id) => {
				const url = assets[id];
				if (!url) throw new Error(`asset missing: ${id}`);
				return url;
			}
		});
		const page = await browser.newPage({ viewport: { width: 1920, height: 1080 } });
		await page.setContent(html, { waitUntil: 'networkidle' });
		await page.evaluate(async () => {
			await document.fonts.ready;
			await Promise.all(
				[...document.images].map(async (image) => {
					if (!image.complete) {
						await new Promise((resolve, reject) => {
							image.addEventListener('load', resolve, { once: true });
							image.addEventListener('error', reject, { once: true });
						});
					}
					if (image.naturalWidth <= 0) throw new Error(`image failed to load: ${image.src}`);
				})
			);
		});
		if (process.argv.includes('--screenshot')) {
			process.stdout.write(await page.screenshot({ type: 'png', fullPage: true }));
		} else {
			const pdf = await page.pdf({
				width: '1920px',
				height: '1080px',
				printBackground: true,
				preferCSSPageSize: true
			});
			process.stdout.write(pdf);
		}
	}
} finally {
	await browser.close();
}
