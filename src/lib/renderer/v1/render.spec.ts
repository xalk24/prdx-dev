import { describe, expect, it } from 'vitest';
import { renderDeckHtml, SCENE_RENDERER_VERSION } from './render';
import type { ContractDeck } from '$lib/editor/contract';
const deck: ContractDeck = {
	schemaVersion: '1.0',
	deckId: 'golden',
	canvas: { width: 1920, height: 1080 },
	theme: {
		version: '1.0',
		palette: { primary: '#2457C5' },
		typography: { heading: 'Inter', body: 'Georgia' },
		background: '#FFFFFF'
	},
	slides: [
		{
			id: 'slide-1',
			name: 'Fidelity',
			elements: [
				{
					id: 'text',
					type: 'text',
					frame: { x: 10, y: 20, width: 500, height: 100 },
					visible: true,
					locked: false,
					content: 'Text <safe>',
					fontFamily: 'Inter',
					fontSize: 72,
					weight: 700,
					color: '#20242B',
					align: 'left',
					lineHeight: 1.1
				},
				{
					id: 'image',
					type: 'image',
					frame: { x: 600, y: 20, width: 400, height: 300 },
					visible: true,
					locked: false,
					assetId: 'asset one',
					fit: 'cover',
					crop: { version: '1.0', x: 0.1, y: 0.2, width: 0.5, height: 0.5 }
				},
				{
					id: 'rect',
					type: 'shape',
					frame: { x: 10, y: 200, width: 100, height: 80 },
					visible: true,
					locked: false,
					shape: 'rect',
					fill: '#2457C5',
					stroke: '#000000',
					strokeWidth: 2,
					opacity: 1
				},
				{
					id: 'ellipse',
					type: 'shape',
					frame: { x: 130, y: 200, width: 80, height: 80 },
					visible: true,
					locked: false,
					shape: 'ellipse',
					fill: '#087E8B',
					stroke: '#000000',
					strokeWidth: 0,
					opacity: 0.8
				},
				{
					id: 'line',
					type: 'shape',
					frame: { x: 230, y: 240, width: 300, height: 2 },
					visible: true,
					locked: false,
					shape: 'line',
					fill: '#000000',
					stroke: '#D92D20',
					strokeWidth: 7,
					opacity: 1
				}
			]
		}
	],
	comments: []
};
describe('scene renderer v1', () => {
	it('renders deterministic print pages for every Deck 1.0 element', () => {
		const html = renderDeckHtml(deck);
		expect(html).toContain(`data-renderer-version="${SCENE_RENDERER_VERSION}"`);
		expect(html).toContain('@page{size:1920px 1080px;margin:0}');
		expect(html).toContain('Text &lt;safe&gt;');
		expect(html).toContain('data-crop-version="1.0"');
		expect(html).toContain('left:-20%;top:-40%;width:200%;height:200%');
		expect(html).toContain('/api/v1/assets/asset%20one');
		expect(html).toContain('data-asset-id="asset one"');
		expect(html).toContain('window.__PRESENTATOR_RENDER_READY__');
		expect(html).toContain('image.naturalWidth>0');
		expect(html).toContain('asset_load_failed:');
		expect(html).toContain("@font-face{font-family:Inter;src:url('data:font/woff2;base64,");
		expect(html).toContain('document.fonts.check');
		expect(html).toContain('font_family_mismatch:');
		expect(html).toContain('scene-shape ellipse');
		expect(html).toContain('scene-shape line');
		expect(html).toContain('height:7px;background:#D92D20');
		expect(html).not.toContain('.scene-shape.line{');
	});
	it('is byte-stable and escapes untrusted content', () => {
		const once = renderDeckHtml(deck);
		expect(renderDeckHtml(deck)).toBe(once);
		expect(once).not.toContain('Text <safe>');
	});
});
