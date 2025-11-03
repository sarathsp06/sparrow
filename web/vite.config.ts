import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';
import devtoolsJson from 'vite-plugin-devtools-json';

import { resolve } from 'path';

export default defineConfig({
	plugins: [
		tailwindcss(),
		sveltekit(),
		devtoolsJson()
	],
	server: {
		allowedHosts:["d534c054bf2f.ngrok-free.app"],
	},
	resolve: {
		alias: {
			'@bufbuild/protobuf': resolve(
				__dirname,
				'./node_modules/@bufbuild/protobuf/dist/esm'
			),
		},
	},
});
