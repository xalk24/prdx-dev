import { defineConfig } from 'vite';
import { resolve } from 'node:path';

export default defineConfig({
	build: {
		emptyOutDir: true,
		outDir: 'build-renderer',
		lib: {
			entry: resolve('src/lib/renderer/v1/render.ts'),
			formats: ['es'],
			fileName: () => 'render.js'
		}
	}
});
