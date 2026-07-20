import type { Deck, DeckElement } from './model';

export interface ContractFrame {
	x: number;
	y: number;
	width: number;
	height: number;
}
export interface ContractElement {
	id: string;
	type: 'text' | 'image' | 'shape';
	frame: ContractFrame;
	visible: boolean;
	locked: boolean;
	content?: string;
	color?: string;
	fill?: string;
}
export interface ContractDeck {
	schemaVersion: '1.0';
	deckId: string;
	canvas: { width: 1920; height: 1080 };
	theme: { background: string };
	slides: Array<{ id: string; name: string; elements: ContractElement[] }>;
	comments: Array<{ id: string; slideId: string; text: string; resolved: boolean }>;
}

const percent = (value: number, total: number) => Number(((value / total) * 100).toFixed(3));

/** Keeps backend-owned Deck 1.0 details at the API edge; the editor uses viewport percentages. */
export function fromContract(source: ContractDeck): Deck {
	return {
		id: source.deckId,
		title: 'Untitled presentation',
		revision: 1,
		theme: 'quiet',
		slides: source.slides.map((slide) => ({
			id: slide.id,
			title: slide.name,
			background: source.theme.background,
			elements: slide.elements.map((element): DeckElement => ({
				id: element.id,
				kind: element.type === 'shape' ? 'shape' : 'text',
				name: element.type[0].toUpperCase() + element.type.slice(1),
				x: percent(element.frame.x, source.canvas.width),
				y: percent(element.frame.y, source.canvas.height),
				w: percent(element.frame.width, source.canvas.width),
				h: percent(element.frame.height, source.canvas.height),
				content: element.content ?? '',
				fill: element.color ?? element.fill ?? '#20242B',
				locked: element.locked,
				hidden: !element.visible
			}))
		}))
	};
}
