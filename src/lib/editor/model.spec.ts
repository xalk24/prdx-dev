import { describe, expect, it } from 'vitest';
import { duplicateSlide, fixtureDeck, updateGeometry } from './model';
describe('editor model', () => {
	it('duplicates without mutating the fixture', () => {
		const next = duplicateSlide(fixtureDeck, 0);
		expect(next.slides).toHaveLength(5);
		expect(next.slides[1].title).toBe('A sharper story copy');
		expect(fixtureDeck.slides).toHaveLength(4);
	});
	it('rejects invalid geometry and clamps values', () => {
		const element = fixtureDeck.slides[0].elements[0];
		expect(updateGeometry(element, 'x', Number.NaN)).toBe(element);
		expect(updateGeometry(element, 'x', -2).x).toBe(0);
		expect(updateGeometry(element, 'w', 0).w).toBe(1);
	});
});
