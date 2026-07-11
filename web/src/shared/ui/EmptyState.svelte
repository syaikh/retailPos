<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { Component } from 'svelte';

  let {
    icon,
    title,
    subtitle,
    action,
    class: className = '',
  }: {
    icon?: Component;
    title: string;
    subtitle?: string;
    action?: Snippet;
    class?: string;
  } = $props();
</script>

<div class="px-4 py-12 text-center {className}">
  {#if icon}
    <div class="w-20 h-20 mx-auto flex justify-center rounded-2xl bg-surface-default">
      <svelte:component this={icon} size={32} class="text-text-muted mt-4" />
    </div>
  {/if}
  <p class="text-text-primary font-semibold mt-4">{title}</p>
  {#if subtitle}
    <p class="text-text-muted text-sm mt-1">{subtitle}</p>
  {/if}
  {#if action}
    <div class="mt-4">
      {@render action()}
    </div>
  {/if}
</div>
