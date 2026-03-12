import staticAdapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	preprocess: vitePreprocess({
		prebundleSvelteLibraries: false,
	}),

	kit: {
		adapter: staticAdapter({
			pages: '../internal/ui/dist',
			assets: '../internal/ui/dist',
			fallback: 'index.html',
			strict: false
		}),
		alias: {
			'@bufbuild/protobuf': './node_modules/@bufbuild/protobuf/dist/esm'
		}
	}
};

export default config;
