import type { ContractDeck, ContractElement } from '$lib/editor/contract';

export const SCENE_RENDERER_VERSION = '1.0.0';
export interface RenderOptions {
	assetUrl?: (assetId: string) => string;
	title?: string;
	slideId?: string;
}
const escape = (value: string) =>
	value
		.replaceAll('&', '&amp;')
		.replaceAll('<', '&lt;')
		.replaceAll('>', '&gt;')
		.replaceAll('"', '&quot;')
		.replaceAll("'", '&#39;');
const css = (value: string) => value.replace(/[^#(),.%\-\w\s]/g, '');
function elementHtml(element: ContractElement, assetUrl: (id: string) => string) {
	if (!element.visible) return '';
	const f = element.frame;
	const frame = `left:${f.x}px;top:${f.y}px;width:${f.width}px;height:${f.height}px`;
	if (element.type === 'text')
		return `<div class="scene-element scene-text" data-element-id="${escape(element.id)}" style="${frame};font-family:${css(element.fontFamily)};font-size:${element.fontSize}px;font-weight:${element.weight};color:${css(element.color)};text-align:${element.align};line-height:${element.lineHeight}">${escape(element.content)}</div>`;
	if (element.type === 'image')
		return `<img class="scene-element scene-image" data-element-id="${escape(element.id)}" style="${frame};object-fit:${element.fit}" src="${escape(assetUrl(element.assetId))}" alt="" />`;
	const shapeClass =
		element.shape === 'ellipse' ? ' ellipse' : element.shape === 'line' ? ' line' : '';
	return `<div class="scene-element scene-shape${shapeClass}" data-element-id="${escape(element.id)}" style="${frame};background:${css(element.fill)};border:${element.strokeWidth}px solid ${css(element.stroke)};opacity:${element.opacity}"></div>`;
}
export function renderDeckHtml(deck: ContractDeck, options: RenderOptions = {}) {
	const assetUrl = options.assetUrl ?? ((id) => `/api/v1/assets/${encodeURIComponent(id)}`);
	const selected = options.slideId
		? deck.slides.filter((slide) => slide.id === options.slideId)
		: deck.slides;
	const slides = selected
		.map(
			(slide, index) =>
				`<section class="scene-page" data-slide-id="${escape(slide.id)}" aria-label="Slide ${index + 1}: ${escape(slide.name)}">${slide.elements.map((e) => elementHtml(e, assetUrl)).join('')}</section>`
		)
		.join('');
	return `<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=1920"><meta name="presentator-renderer" content="${SCENE_RENDERER_VERSION}"><title>${escape(options.title ?? 'Presentator deck')}</title><style>@page{size:1920px 1080px;margin:0}*{box-sizing:border-box}html,body{margin:0;padding:0;background:${css(deck.theme.background)}}body{font-family:${css(deck.theme.typography.body)},sans-serif}.scene-page{position:relative;width:1920px;height:1080px;overflow:hidden;background:${css(deck.theme.background)};break-after:page;page-break-after:always}.scene-page:last-child{break-after:auto;page-break-after:auto}.scene-element{position:absolute;margin:0;padding:0}.scene-text{white-space:pre-wrap;overflow:hidden}.scene-image{display:block}.scene-shape{display:block}.scene-shape.ellipse{border-radius:50%}.scene-shape.line{height:0!important;background:transparent!important;border-width:0!important;border-top:${1}px solid currentColor!important}</style></head><body data-renderer-version="${SCENE_RENDERER_VERSION}">${slides}</body></html>`;
}
