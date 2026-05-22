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
   <!-- Sidebar -->
   <Sidebar bind:currentPath />

   <!-- Main column -->
   <div class="flex flex-col flex-1 min-w-0 overflow-hidden">
     <Topbar {currentPath} />

     <!-- Scrollable content area -->
     <main class="flex-1 overflow-y-auto bg-bg-default grid">
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
