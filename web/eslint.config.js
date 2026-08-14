import js from "@eslint/js";
import svelte from "eslint-plugin-svelte";
import globals from "globals";
import svelteConfig from "./svelte.config.js";

export default [
  js.configs.recommended,
  ...svelte.configs.recommended,
  {
    languageOptions: {
      globals: { ...globals.browser, ...globals.node },
    },
  },
  {
    // Svelte files and rune modules are parsed by svelte-eslint-parser, which
    // needs the project's svelte.config.js to resolve aliases and runes.
    files: ["**/*.svelte", "**/*.svelte.js"],
    languageOptions: {
      parserOptions: { svelteConfig },
    },
  },
  { ignores: ["dist/", ".svelte-kit/", "node_modules/"] },
];
