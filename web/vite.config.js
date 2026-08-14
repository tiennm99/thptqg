import { sveltekit } from "@sveltejs/kit/vite";
import tailwindcss from "@tailwindcss/vite";
import { defineConfig } from "vitest/config";

export default defineConfig({
  plugins: [tailwindcss(), sveltekit()],
  test: {
    // Only the framework-free modules are unit-tested: score tiers, query-mode
    // classification and the ASCII fold that has to match the Go parser.
    include: ["src/lib/**/*.test.js"],
  },
});
