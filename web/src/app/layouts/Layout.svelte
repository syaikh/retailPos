<script lang="ts">
  import type { Snippet } from 'svelte';
  import Sidebar from '$app/layouts/Sidebar.svelte';
  import Topbar from '$app/layouts/Topbar.svelte';
  import { fade, fly } from 'svelte/transition';

  let {
    currentPath = '/',
    children
  }: {
    currentPath?: string;
    children: Snippet;
  } = $props();

  let isMobileMenuOpen = $state(false);

  function closeMobileMenu() {
    isMobileMenuOpen = false;
  }
</script>

<div class="flex h-screen bg-bg-default">
   <!-- Skip-to-content for keyboard users -->
   <a href="#main-content" class="sr-only focus:not-sr-only focus:fixed focus:top-4 focus:left-4 focus:z-[9999] focus:px-4 focus:py-2 focus:bg-primary-default focus:text-white focus:rounded-xl focus:text-sm focus:font-semibold focus:outline-none focus:ring-2 focus:ring-primary-light focus:shadow-glow-primary">
     Skip to main content
   </a>

   <!-- Mobile backdrop -->
   {#if isMobileMenuOpen}
     <div
       class="fixed inset-0 bg-black/50 z-40 md:hidden"
       onclick={closeMobileMenu}
       transition:fade={{ duration: 200 }}
       role="presentation"
     ></div>
   {/if}

   <!-- Sidebar -->
   <Sidebar bind:currentPath {isMobileMenuOpen} onclose={closeMobileMenu} />

   <!-- Main column -->
   <div class="flex flex-col flex-1 min-w-0 overflow-hidden">
     <Topbar {currentPath} ontogglemenu={() => { isMobileMenuOpen = !isMobileMenuOpen; }} />

     <!-- Scrollable content area -->
     <main id="main-content" class="flex-1 overflow-y-auto bg-bg-default grid" style="scrollbar-gutter: stable;">
       {#key currentPath}
         <div
           in:fly={{ y: 15, duration: 300, delay: 150 }}
           out:fade={{ duration: 150 }}
           style="grid-area: 1 / 1 / 2 / 2"
           class="p-4 md:p-6"
         >
           {@render children()}
         </div>
       {/key}
    </main>
  </div>
</div>
