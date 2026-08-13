import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// One build serves every page. The app resolves which dataset to show from the
// URL (src/router.js), so there are no per-dataset build variants.
//
// `base` is absolute, which is what lets the single emitted index.html work as
// an entry point at any depth: it references /thptqg/assets/... regardless of
// the directory it is served from. The deploy step copies it to each dataset
// path, so every URL is a real static file and no SPA 404-fallback is needed —
// and the existing ?q= deep links keep working, which that fallback would break.
//
// publicDir holds only the gzipped databases, staged there by the assembler
// (assembler/internal/databases). Nothing uncompressed is ever placed in it.
//
// It sits at the repository root rather than inside this workspace because
// parser writes it, so the path has to climb out of web/. Vite resolves
// publicDir against the project root, which is this directory.
//
// outDir is left at its default, so the build lands in web/dist and this
// workspace stays self-contained. The Pages artifact (_site) is assembled at
// the repository root by the assembler (assembler/internal/site), since that is
// where the deploy action uploads from.
export default defineConfig({
  plugins: [react()],
  base: "/thptqg/",
  publicDir: "../.build/public",
});
