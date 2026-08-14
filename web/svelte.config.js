import adapter from "@sveltejs/adapter-static";
import { vitePreprocess } from "@sveltejs/vite-plugin-svelte";

/**
 * Every route is prerendered to a real HTML file, so GitHub Pages serves static
 * documents and no SPA 404-fallback is needed — a fallback would rewrite the URL
 * and break the ?q= deep links.
 *
 * `paths.relative: false` keeps every asset URL absolute (/thptqg/_app/...), so
 * the copy of index.html that serves as 404.html resolves its assets from any
 * depth — GitHub Pages answers /thptqg/anything/at/all/ with that one file.
 *
 * `files.assets` points outside this workspace because the assembler stages the
 * gzipped databases there (assembler/internal/databases).
 *
 * @type {import('@sveltejs/kit').Config}
 */
export default {
  preprocess: vitePreprocess(),
  kit: {
    adapter: adapter({ pages: "dist", assets: "dist", strict: true }),
    paths: { base: "/thptqg", relative: false },
    files: { assets: "../.build/public" },
  },
};
