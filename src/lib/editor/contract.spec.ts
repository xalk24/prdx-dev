import { describe, expect, it } from 'vitest';
import { fromContract, type ContractDeck } from './contract';

describe('Deck 1.0 adapter', () => {
	it('maps 1920×1080 contract frames into editor percentages', () => {
		const source: ContractDeck = {
			schemaVersion: '1.0',
			deckId: 'deck-demo',
			canvas: { width: 1920, height: 1080 },
			theme: { background: '#FFFFFF' },
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
							color: '#111827'
						}
					]
				}
			],
			comments: []
		};
		const deck = fromContract(source);
		const element = deck.slides[0].elements[0];
		expect(deck.id).toBe('deck-demo');
		expect(element).toMatchObject({
			x: 10,
			y: 10,
			w: 50,
			h: 50,
			content: 'Quiet Precision',
			hidden: false
		});
	});
});
