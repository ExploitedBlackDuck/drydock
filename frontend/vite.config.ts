import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';

// Vite builds the Svelte app into dist/, which is embedded into the Go binary
// (frontend/embed.go) and served by the Wails asset server.
export default defineConfig({
  plugins: [svelte()],
  build: {
    outDir: 'dist',
    // Keep the committed dist/.gitkeep so the Go embed (frontend/embed.go) has a
    // directory to embed on a fresh checkout, before the frontend is ever built.
    // Emptying would delete it and break `go build` on a bare machine.
    emptyOutDir: false,
    target: 'esnext',
  },
});
