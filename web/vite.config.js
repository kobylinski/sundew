import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';

export default defineConfig({
  plugins: [svelte()],
  base: '/',
  build: { outDir: '../internal/ui/static', emptyOutDir: false },
  server: { proxy: { '/api': 'http://127.0.0.1:8025' } },
});
