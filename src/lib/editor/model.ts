export type JobState = 'idle' | 'queued' | 'running' | 'succeeded' | 'failed';
export type ElementKind = 'text' | 'shape';
export interface DeckElement {
	id: string;
	kind: ElementKind;
	name: string;
	x: number;
	y: number;
	w: number;
	h: number;
	content: string;
	fill: string;
	locked?: boolean;
	hidden?: boolean;
}
export interface Slide {
	id: string;
	title: string;
	background: string;
	elements: DeckElement[];
}
export interface Deck {
	id: string;
	title: string;
	revision: number;
	theme: 'quiet' | 'midnight';
	slides: Slide[];
}

export const fixtureDeck: Deck = {
	id: 'deck-quarterly-story',
	title: 'Quarterly story',
	revision: 12,
	theme: 'quiet',
	slides: [
		{
			id: 'slide-1',
			title: 'A sharper story',
			background: '#f7f3e9',
			elements: [
				{
					id: 'eyebrow',
					kind: 'text',
					name: 'Eyebrow',
					x: 9,
					y: 13,
					w: 32,
					h: 6,
					content: 'QUARTERLY BRIEF · 2026',
					fill: '#087e8b'
				},
				{
					id: 'headline',
					kind: 'text',
					name: 'Headline',
					x: 9,
					y: 25,
					w: 57,
					h: 25,
					content: 'From signal\nto shared direction.',
					fill: '#20242b'
				},
				{
					id: 'body',
					kind: 'text',
					name: 'Supporting copy',
					x: 9,
					y: 58,
					w: 47,
					h: 14,
					content: 'A concise view of the decisions, momentum and next moves shaping this quarter.',
					fill: '#62666d'
				},
				{
					id: 'accent',
					kind: 'shape',
					name: 'Signal card',
					x: 70,
					y: 15,
					w: 22,
					h: 68,
					content: '64%',
					fill: '#2457c5'
				}
			]
		},
		{
			id: 'slide-2',
			title: 'Three signals',
			background: '#fcfbf8',
			elements: [
				{
					id: 'title-2',
					kind: 'text',
					name: 'Headline',
					x: 9,
					y: 12,
					w: 65,
					h: 14,
					content: 'Three signals worth acting on',
					fill: '#20242b'
				}
			]
		},
		{
			id: 'slide-3',
			title: 'Momentum',
			background: '#e9f1ee',
			elements: [
				{
					id: 'title-3',
					kind: 'text',
					name: 'Headline',
					x: 9,
					y: 12,
					w: 62,
					h: 14,
					content: 'Momentum, made visible',
					fill: '#20242b'
				}
			]
		},
		{
			id: 'slide-4',
			title: 'Next moves',
			background: '#20242b',
			elements: [
				{
					id: 'title-4',
					kind: 'text',
					name: 'Headline',
					x: 9,
					y: 12,
					w: 70,
					h: 14,
					content: 'The next moves are clear',
					fill: '#fff'
				}
			]
		}
	]
};
export const cloneDeck = (deck: Deck): Deck => structuredClone(deck);
export function updateGeometry(element: DeckElement, key: 'x' | 'y' | 'w' | 'h', value: number) {
	if (!Number.isFinite(value)) return element;
	return { ...element, [key]: Math.max(key === 'w' || key === 'h' ? 1 : 0, Math.min(100, value)) };
}
export function duplicateSlide(deck: Deck, index: number): Deck {
	const source = deck.slides[index];
	if (!source) return deck;
	const copy = structuredClone(source);
	copy.id = `${source.id}-copy-${deck.slides.length + 1}`;
	copy.title = `${source.title} copy`;
	return {
		...deck,
		slides: [...deck.slides.slice(0, index + 1), copy, ...deck.slides.slice(index + 1)]
	};
}
