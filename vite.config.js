import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// VARIANT selects which dataset to ship:
//   (unset)  → main site at /thptqg2017/ using public/
//   old      → at /thptqg2017/old/ using public-old/
//   old2     → at /thptqg2017/old2/ using public-old2/
const VARIANT = process.env.VARIANT || "";
const VARIANT_CONFIG = {
  "":     { base: "/thptqg/2017/",      publicDir: "public",      outDir: "dist" },
  old:    { base: "/thptqg/2017/old/",  publicDir: "public-old",  outDir: "dist/old" },
  old2:   { base: "/thptqg/2017/old2/", publicDir: "public-old2", outDir: "dist/old2" },
};
const cfg = VARIANT_CONFIG[VARIANT];
if (!cfg) throw new Error(`Unknown VARIANT: ${VARIANT}`);

export default defineConfig({
  plugins: [react()],
  base: cfg.base,
  publicDir: cfg.publicDir,
  build: {
    outDir: cfg.outDir,
    emptyOutDir: VARIANT === "",  // only main build wipes dist root
  },
});
