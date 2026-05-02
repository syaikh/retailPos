<script lang="ts">
  import type { Snippet } from 'svelte';
  import Sidebar from '$lib/components/Sidebar.svelte';
  import Topbar from '$lib/components/Topbar.svelte';

  let {
    currentPath = '/',
    children
  }: {
    currentPath?: string;
    children: Snippet;
  } = $props();
</script>

<div class="flex h-screen overflow-hidden bg-bg">
  <!-- Sidebar -->
  <Sidebar bind:currentPath />

  <!-- Main column -->
  <div class="flex flex-col flex-1 min-w-0 overflow-hidden">
    <Topbar {currentPath} />

    <!-- Scrollable content area -->
    <main class="flex-1 overflow-y-auto bg-bg p-6">
      <div class="animate-slide-up">
        {@render children()}
      </div>
    </main>
  </div>
</div>
