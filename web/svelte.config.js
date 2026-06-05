import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

export default {
  preprocess: vitePreprocess(),
  compilerOptions: {
    warningFilter: (warning) => {
      // Suppress unused CSS selector warnings for print styles
      if (warning.code === 'css_unused_selector') {
        return false;
      }
      return true;
    }
  }
};