import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [svelte(), tailwindcss()],
  build: {
    outDir: '../internal/uiserver/dist',
    emptyOutDir: true,
  },
  server: { port: 8081 },
})