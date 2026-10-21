import js from '@eslint/js';
import tseslint from 'typescript-eslint';
import svelte from 'eslint-plugin-svelte';
import globals from 'globals';
import { readdirSync } from 'node:fs';

// Import-boundary rules (docs/adr/ADR_Frontend_Module_Import_Boundaries.md).
// For each module X, forbid deep paths into OTHER modules while allowing the
// bare barrel `$modules/<name>` and X's own internals. Generated from the
// directory listing so a new module needs no config change. src/app/ is
// exempt (composition root). src/shared/ may not import $modules at all.
const moduleNames = readdirSync(new URL('./src/modules/', import.meta.url), {
  withFileTypes: true,
})
  .filter((entry) => entry.isDirectory())
  .map((entry) => entry.name);

const importBoundaryConfigs = moduleNames.map((name) => ({
  files: [`src/modules/${name}/**/*.{ts,svelte}`],
  rules: {
    'no-restricted-imports': [
      'warn',
      {
        patterns: [
          {
            // Any deep path ($modules/<other>/...) into another module.
            // Bare barrels ($modules/<other>) and own-module paths are allowed.
            regex: `^\\$modules/(?!${name}(\\/|$))[^/]+\\/`,
            message:
              'Cross-module imports must use the module barrel ($modules/<name>), never a deep path into services/, types/, components/ or lib/.',
          },
        ],
      },
    ],
  },
}));

const sharedNoModuleImports = {
  files: ['src/shared/**/*.{ts,svelte}'],
  rules: {
    'no-restricted-imports': [
      'warn',
      {
        patterns: [
          {
            regex: '^\\$modules/',
            message:
              'Shared code must not import from $modules (dependency inversion). Move the code to $shared instead.',
          },
        ],
      },
    ],
  },
};

export default tseslint.config(
  js.configs.recommended,
  ...tseslint.configs.recommended,
  ...svelte.configs['flat/recommended'],
  {
    rules: {
      '@typescript-eslint/no-explicit-any': 'warn',
      '@typescript-eslint/no-unused-vars': [
        'warn',
        {
          argsIgnorePattern: '^_',
          varsIgnorePattern: '^_',
          caughtErrorsIgnorePattern: '^_',
        },
      ],
      '@typescript-eslint/no-unused-expressions': 'warn',
      '@typescript-eslint/no-require-imports': 'off',
      'svelte/require-each-key': 'warn',
      'svelte/prefer-svelte-reactivity': 'warn',
      'svelte/no-useless-children-snippet': 'warn',
      'svelte/no-unused-svelte-ignore': 'warn',
      'no-useless-catch': 'warn',
      'no-unsafe-finally': 'warn',
      'no-case-declarations': 'warn',
      'prefer-const': 'warn',
      'no-undef': 'off',
    },
  },
  {
    languageOptions: {
      globals: {
        ...globals.browser,
        ...globals.node,
      },
    },
  },
  {
    files: ['**/*.svelte'],
    languageOptions: {
      parserOptions: {
        parser: tseslint.parser,
      },
    },
    rules: {
      'prefer-const': 'off',
    },
  },
  {
    files: ['**/*.svelte.ts'],
    languageOptions: {
      parserOptions: {
        parser: tseslint.parser,
      },
    },
  },
  ...importBoundaryConfigs,
  sharedNoModuleImports,
  {
    ignores: ['dist/', 'build/', 'node_modules/', '*.config.js', '*.config.ts'],
  },
);
