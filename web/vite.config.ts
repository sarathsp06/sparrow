import devtoolsJson from 'vite-plugin-devtools-json';
import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

import { resolve } from 'path';

export default defineConfig({
	plugins: [
		tailwindcss(),
		sveltekit(),
		devtoolsJson()
	],
	resolve: {
		alias: {
			'@bufbuild/protobuf': resolve(
				__dirname,
				'./node_modules/@bufbuild/protobuf/dist/esm'
			),
		},
	},
});
