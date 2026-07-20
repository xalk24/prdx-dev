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
			if (!window.__PRESENTATOR_RENDER_READY__)
				throw new Error('renderer readiness contract missing');
			await window.__PRESENTATOR_RENDER_READY__;
		});
		if (process.argv.includes('--inspect')) {
			const fidelity = await page.evaluate(() => {
				const text = document.querySelector('[data-element-id="text"]');
				const line = document.querySelector('[data-element-id="line"]');
				const crop = document.querySelector('[data-crop-version]');
				const image = crop?.querySelector('img');
				const textStyle = text ? getComputedStyle(text) : null;
				const lineStyle = line ? getComputedStyle(line) : null;
				const imageStyle = image ? getComputedStyle(image) : null;
				return {
					fontFamily: textStyle?.fontFamily.split(',')[0].replace(/["']/g, '').trim(),
					textHeight: text?.getBoundingClientRect().height,
					lineHeight: textStyle ? Number.parseFloat(textStyle.lineHeight) : 0,
					lineColor: lineStyle?.backgroundColor,
					lineWidth: lineStyle?.height,
					cropVersion: crop?.getAttribute('data-crop-version'),
					cropLeft: imageStyle?.left,
					cropWidth: imageStyle?.width
				};
			});
			process.stdout.write(JSON.stringify(fidelity));
		} else if (process.argv.includes('--screenshot')) {
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
