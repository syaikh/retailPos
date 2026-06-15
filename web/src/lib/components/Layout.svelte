<script lang="ts">
  import type { Snippet } from 'svelte';
  import Sidebar from '$lib/components/Sidebar.svelte';
  import Topbar from '$lib/components/Topbar.svelte';
  import { fade, fly } from 'svelte/transition';

  let {
    currentPath = '/',
    children
  }: {
    currentPath?: string;
    children: Snippet;
  } = $props();
</script>

<div class="flex h-screen bg-bg-default">
   <!-- Skip-to-content for keyboard users -->
   <a href="#main-content" class="sr-only focus:not-sr-only focus:fixed focus:top-4 focus:left-4 focus:z-[9999] focus:px-4 focus:py-2 focus:bg-primary-default focus:text-white focus:rounded-xl focus:text-sm focus:font-semibold focus:outline-none focus:ring-2 focus:ring-primary-light focus:shadow-glow-primary">
     Skip to main content
   </a>

   <!-- Sidebar -->
   <Sidebar bind:currentPath />

   <!-- Main column -->
   <div class="flex flex-col flex-1 min-w-0 overflow-hidden">
     <Topbar {currentPath} />

     <!-- Scrollable content area -->
     <main id="main-content" class="flex-1 overflow-y-auto bg-bg-default grid" style="scrollbar-gutter: stable;">
      {#key currentPath}
        <div 
          in:fly={{ y: 15, duration: 300, delay: 150 }} 
          out:fade={{ duration: 150 }} 
          style="grid-area: 1 / 1 / 2 / 2" 
          class="p-6"
        >
          {@render children()}
        </div>
      {/key}
    </main>
  </div>
</div>
