import type { Deck, DeckElement } from './model';
export interface Frame {
	x: number;
	y: number;
	width: number;
	height: number;
}
interface BaseElement {
	id: string;
	type: 'text' | 'image' | 'shape';
	frame: Frame;
	visible: boolean;
	locked: boolean;
}
export interface TextElement extends BaseElement {
	type: 'text';
	content: string;
	fontFamily: 'Inter' | 'Arial' | 'Georgia';
	fontSize: number;
	weight: 400 | 500 | 600 | 700;
	color: string;
	align: 'left' | 'center' | 'right';
	lineHeight: number;
}
export interface ImageElement extends BaseElement {
	type: 'image';
	assetId: string;
	fit: 'contain' | 'cover';
	crop?: {
		version: '1.0';
		x: number;
		y: number;
		width: number;
		height: number;
	};
}
export interface ShapeElement extends BaseElement {
	type: 'shape';
	shape: 'rect' | 'ellipse' | 'line';
	fill: string;
	stroke: string;
	strokeWidth: number;
	opacity: number;
}
export type ContractElement = TextElement | ImageElement | ShapeElement;
export interface ContractDeck {
	schemaVersion: '1.0';
	deckId: string;
	canvas: { width: 1920; height: 1080 };
	theme: {
		version: '1.0';
		palette: Record<string, string>;
		typography: { heading: string; body: string };
		background: string;
	};
	slides: Array<{ id: string; name: string; elements: ContractElement[] }>;
	comments: Array<{
		id: string;
		slideId: string;
		anchor?: { x: number; y: number };
		text: string;
		resolved: boolean;
	}>;
}
const percent = (v: number, total: number) => (v / total) * 100;
const pixels = (v: number, total: number) => Number(((v / 100) * total).toFixed(6));
function newElement(item: DeckElement): ContractElement {
	const common = {
		id: item.id,
		frame: { x: 0, y: 0, width: 1, height: 1 },
		visible: true,
		locked: false
	};
	if (item.kind === 'shape')
		return {
			...common,
			type: 'shape',
			shape: 'rect',
			fill: item.fill,
			stroke: item.fill,
			strokeWidth: 0,
			opacity: 1
		};
	if (item.kind === 'image')
		return { ...common, type: 'image', assetId: item.content, fit: 'contain' };
	return {
		...common,
		type: 'text',
		content: item.content,
		fontFamily: 'Inter',
		fontSize: 32,
		weight: 400,
		color: item.fill,
		align: 'left',
		lineHeight: 1.2
	};
}
export function fromContract(source: ContractDeck, revision = 1): Deck {
	return {
		id: source.deckId,
		title: 'Presentator deck',
		revision,
		theme: 'quiet',
		slides: source.slides.map((slide) => ({
			id: slide.id,
			title: slide.name,
			background: source.theme.background,
			elements: slide.elements.map((e): DeckElement => ({
				id: e.id,
				kind: e.type,
				name: e.type[0].toUpperCase() + e.type.slice(1),
				x: percent(e.frame.x, source.canvas.width),
				y: percent(e.frame.y, source.canvas.height),
				w: percent(e.frame.width, source.canvas.width),
				h: percent(e.frame.height, source.canvas.height),
				content: e.type === 'text' ? e.content : e.type === 'image' ? e.assetId : '',
				fill: e.type === 'text' ? e.color : e.type === 'shape' ? e.fill : '#20242B',
				locked: e.locked,
				hidden: !e.visible
			}))
		}))
	};
}
/** Patch editor-owned fields onto a clone, preserving every unsupported Deck 1.0 field. */
export function toContract(editor: Deck, canonical: ContractDeck): ContractDeck {
	const next = structuredClone(canonical);
	next.deckId = editor.id;
	next.slides = editor.slides.map((slide) => {
		const old = next.slides.find((s) => s.id === slide.id);
		return {
			id: slide.id,
			name: slide.title,
			elements: slide.elements.map((item) => {
				const existing = old?.elements.find((e) => e.id === item.id);
				const base: ContractElement = existing ?? newElement(item);
				base.frame = {
					x: pixels(item.x, next.canvas.width),
					y: pixels(item.y, next.canvas.height),
					width: pixels(item.w, next.canvas.width),
					height: pixels(item.h, next.canvas.height)
				};
				base.visible = !item.hidden;
				base.locked = !!item.locked;
				if (base.type === 'text') {
					base.content = item.content;
					base.color = item.fill;
				}
				if (base.type === 'shape') base.fill = item.fill;
				return base as ContractElement;
			})
		};
	});
	return next;
}
