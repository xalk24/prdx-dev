import { describe, expect, it } from 'vitest';
import { fromContract, toContract, type ContractDeck } from './contract';
const fixture: ContractDeck = {
	schemaVersion: '1.0',
	deckId: 'deck-demo',
	canvas: { width: 1920, height: 1080 },
	theme: {
		version: '1.0',
		palette: { primary: '#2563EB' },
		typography: { heading: 'Inter', body: 'Georgia' },
		background: '#FFFFFF'
	},
	slides: [
		{
			id: 'slide-1',
			name: 'Opening',
			elements: [
				{
					id: 'text-1',
					type: 'text',
					frame: { x: 192, y: 108, width: 960, height: 540 },
					visible: true,
					locked: false,
					content: 'Quiet Precision',
					fontFamily: 'Georgia',
					fontSize: 72,
					weight: 700,
					color: '#111827',
					align: 'right',
					lineHeight: 1.1
				},
				{
					id: 'image-1',
					type: 'image',
					frame: { x: 0, y: 0, width: 200, height: 200 },
					visible: true,
					locked: false,
					assetId: 'asset-9',
					fit: 'cover'
				},
				{
					id: 'shape-1',
					type: 'shape',
					frame: { x: 40, y: 40, width: 80, height: 80 },
					visible: true,
					locked: false,
					shape: 'ellipse',
					fill: '#2563EB',
					stroke: '#000000',
					strokeWidth: 3,
					opacity: 0.5
				}
			]
		}
	],
	comments: [
		{ id: 'c1', slideId: 'slide-1', anchor: { x: 0.5, y: 0.6 }, text: 'Refine', resolved: false }
	]
};
describe('lossless Deck 1.0 adapter', () => {
	it('maps server revision and 1920×1080 frames', () => {
		const deck = fromContract(fixture, 12);
		expect(deck.revision).toBe(12);
		expect(deck.slides[0].elements[0]).toMatchObject({ x: 10, y: 10, w: 50, h: 50 });
		expect(deck.slides[0].elements[1].kind).toBe('image');
	});
	it('round-trips unsupported fields losslessly while applying edits', () => {
		const editor = fromContract(fixture, 12);
		editor.slides[0].elements[0].content = 'Changed';
		editor.slides[0].elements[0].x = 20;
		const result = toContract(editor, fixture);
		expect(result.theme).toEqual(fixture.theme);
		expect(result.comments).toEqual(fixture.comments);
		expect(result.slides[0].elements[1]).toEqual(fixture.slides[0].elements[1]);
		expect(result.slides[0].elements[2]).toEqual(fixture.slides[0].elements[2]);
		expect(result.slides[0].elements[0]).toMatchObject({
			content: 'Changed',
			fontFamily: 'Georgia',
			align: 'right',
			frame: { x: 384, y: 108, width: 960, height: 540 }
		});
	});
});
