import js from '@eslint/js';
import ts from 'typescript-eslint';
import svelte from 'eslint-plugin-svelte';
import svelteParser from 'svelte-eslint-parser';
import globals from 'globals';

export default ts.config(
  js.configs.recommended,
  ...ts.configs.recommended,
  ...svelte.configs['flat/recommended'],
  {
    files: ['**/*.svelte'],
    languageOptions: {
      parser: svelteParser,
      parserOptions: {
        parser: ts.parser,
      },
    },
  },
  {
    languageOptions: {
      globals: {
        ...globals.browser,
        // Svelte compiler type macros (not runtime values).
        $$Generic: 'readonly',
        $$Props: 'readonly',
        $$Slots: 'readonly',
      },
    },
  },
  {
    // Generated Wails bindings are not ours to lint.
    ignores: ['dist/', 'wailsjs/', 'node_modules/'],
  },
);
