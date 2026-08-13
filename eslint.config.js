import js from '@eslint/js'
import globals from 'globals'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import { defineConfig, globalIgnores } from 'eslint/config'

export default defineConfig([
  // '**/dist', not 'dist': the build output moved to web/dist when the app
  // became a workspace, and a root-anchored pattern would stop matching it.
  globalIgnores(['**/dist', '.build', '_site']),
  {
    files: ['**/*.{js,jsx}'],
    extends: [
      js.configs.recommended,
      reactHooks.configs.flat.recommended,
      reactRefresh.configs.vite,
    ],
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
      parserOptions: {
        ecmaVersion: 'latest',
        ecmaFeatures: { jsx: true },
        sourceType: 'module',
      },
    },
    rules: {
      'no-unused-vars': ['error', { varsIgnorePattern: '^[A-Z_]' }],
    },
  },
  {
    // Node-executed files (Vite config, site assembly, parser tooling) run with
    // Node globals. The crawler is Go, so it has nothing here.
    files: ['web/vite.config.js', 'web/scripts/**/*.js', 'go-parser/scripts/**/*.js'],
    languageOptions: {
      globals: { ...globals.node },
    },
  },
])
