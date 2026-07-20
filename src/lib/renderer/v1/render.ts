import type { ContractDeck, ContractElement } from '$lib/editor/contract';
import interWoff2 from '@fontsource-variable/inter/files/inter-latin-wght-normal.woff2?inline';

export const SCENE_RENDERER_VERSION = '1.1.0';
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
const font = (value: string) => (value === 'Inter' ? 'Inter' : css(value));
function elementHtml(element: ContractElement, assetUrl: (id: string) => string) {
	if (!element.visible) return '';
	const f = element.frame;
	const frame = `left:${f.x}px;top:${f.y}px;width:${f.width}px;height:${f.height}px`;
	if (element.type === 'text')
		return `<div class="scene-element scene-text" data-element-id="${escape(element.id)}" data-font-family="${escape(element.fontFamily)}" style="${frame};font-family:${font(element.fontFamily)};font-size:${element.fontSize}px;font-weight:${element.weight};color:${css(element.color)};text-align:${element.align};line-height:${element.lineHeight}">${escape(element.content)}</div>`;
	if (element.type === 'image') {
		const image = `<img class="scene-image" data-element-id="${escape(element.id)}" data-asset-id="${escape(element.assetId)}" src="${escape(assetUrl(element.assetId))}" alt="" />`;
		if (!element.crop)
			return image.replace(
				'class="scene-image"',
				`class="scene-element scene-image" style="${frame};object-fit:${element.fit}"`
			);
		const crop = element.crop;
		const imageStyle = `left:${(-crop.x / crop.width) * 100}%;top:${(-crop.y / crop.height) * 100}%;width:${100 / crop.width}%;height:${100 / crop.height}%`;
		return `<div class="scene-element scene-image-crop" data-element-id="${escape(element.id)}" data-crop-version="${crop.version}" style="${frame}">${image.replace('class="scene-image"', `class="scene-image" style="${imageStyle}"`)}</div>`;
	}
	if (element.shape === 'line')
		return `<div class="scene-element scene-shape line" data-element-id="${escape(element.id)}" style="${frame};height:${element.strokeWidth}px;background:${css(element.stroke)};opacity:${element.opacity}"></div>`;
	const shapeClass = element.shape === 'ellipse' ? ' ellipse' : '';
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
	return `<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=1920"><meta name="presentator-renderer" content="${SCENE_RENDERER_VERSION}"><title>${escape(options.title ?? 'Presentator deck')}</title><style>@font-face{font-family:Inter;src:url('${interWoff2}') format('woff2');font-style:normal;font-weight:100 900;font-display:block}@page{size:1920px 1080px;margin:0}*{box-sizing:border-box}html,body{margin:0;padding:0;background:${css(deck.theme.background)}}body{font-family:${font(deck.theme.typography.body)},sans-serif}.scene-page{position:relative;width:1920px;height:1080px;overflow:hidden;background:${css(deck.theme.background)};break-after:page;page-break-after:always}.scene-page:last-child{break-after:auto;page-break-after:auto}.scene-element{position:absolute;margin:0;padding:0}.scene-text{white-space:pre-wrap;overflow:hidden}.scene-image{display:block}.scene-image-crop{overflow:hidden}.scene-image-crop>.scene-image{position:absolute;max-width:none;max-height:none}.scene-shape{display:block}.scene-shape.ellipse{border-radius:50%}</style></head><body data-renderer-version="${SCENE_RENDERER_VERSION}">${slides}<script>window.__PRESENTATOR_RENDER_READY__=(async()=>{await document.fonts.ready;const fontFamilies=[...new Set([...document.querySelectorAll('[data-font-family]')].map(element=>element.dataset.fontFamily).filter(Boolean))];await Promise.all(fontFamilies.map(async family=>{await document.fonts.load('16px "'+family+'"');if(!document.fonts.check('16px "'+family+'"'))throw new Error('font_load_failed:'+family);const elements=[...document.querySelectorAll('[data-font-family="'+CSS.escape(family)+'"]')];if(elements.some(element=>getComputedStyle(element).fontFamily.split(',')[0].replace(/["']/g,'').trim()!==family))throw new Error('font_family_mismatch:'+family)}));const images=[...document.images];await Promise.all(images.map(image=>new Promise((resolve,reject)=>{const fail=()=>reject(new Error('asset_load_failed:'+(image.dataset.assetId||image.dataset.elementId||'unknown')));const verify=()=>image.naturalWidth>0?resolve():fail();if(image.complete){verify();return}image.addEventListener('load',verify,{once:true});image.addEventListener('error',fail,{once:true})})));return Object.freeze({rendererVersion:'${SCENE_RENDERER_VERSION}',assetCount:images.length,fontFamilies})})()</script></body></html>`;
}
