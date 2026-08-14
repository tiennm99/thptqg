import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// One build serves every page; src/router.js resolves the dataset from the URL.
//
// `base` must stay absolute. It makes the single emitted index.html work as an
// entry point at any depth, because asset URLs read /thptqg/assets/... no matter
// which directory serves the page. The deploy step copies that file to every
// dataset path, so each URL is a real static file and needs no SPA 404-fallback
// — a fallback would break the ?q= deep links.
//
// publicDir points outside this workspace because the assembler stages the
// gzipped databases there (assembler/internal/databases). Vite resolves the path
// against the project root, which is this directory.
export default defineConfig({
  plugins: [react()],
  base: "/thptqg/",
  publicDir: "../.build/public",
});
