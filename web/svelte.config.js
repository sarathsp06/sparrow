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
		})
	}
};

export default config;
