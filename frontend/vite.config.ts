import { defineConfig } from 'vite';

// outDir is ../dist-frontend (repo root) so the Go embed in
// frontenddist.go can serve it and Wails can bundle it (see docs/cs/03).
export default defineConfig({
  root: '.',
  clearScreen: false,
  build: {
    outDir: '../dist-frontend',
    emptyOutDir: true,
  },
  server: {
    port: 1420,
    strictPort: true,
  },
});
