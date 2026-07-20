import { readFile } from 'node:fs/promises';
import { describe, expect, it } from 'vitest';

describe('production same-origin packaging', () => {
	it('builds a static SPA and keeps API routing relative to the current origin', async () => {
		const [config, client] = await Promise.all([
			readFile('vite.config.ts', 'utf8'),
			readFile('src/lib/editor/api.ts', 'utf8')
		]);
		expect(config).toContain('@sveltejs/adapter-static');
		expect(config).toContain("fallback: 'index.html'");
		expect(client).toContain("private base = '/api/v1'");
		expect(client).not.toMatch(/https?:\/\/[^']+\/api\/v1/);
	});
});
