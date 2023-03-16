import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [sveltekit()],
	// Define the proxy configuration to communicate with the server.
	server: {
		proxy: {
			'/api': {
				// FIXME: Add Production endpoint for server
				target: process.env.NODE_ENV === 'development' ? 'http://0.0.0.0:3000/api' : '',
				changeOrigin: true,
				// FIXME: Verify that the endpoints don't need the api prefix.
				rewrite: (path: string) => path.replace(/^\/api/, '')
			}
		}
	}
});
