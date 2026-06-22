<script lang="ts">
  let {
    width = 'w-full',
    height = 'h-4',
    rounded = 'rounded-lg',
    class: className = '',
  }: {
    width?: string;
    height?: string;
    rounded?: string;
    class?: string;
  } = $props();
</script>

<div class="skeleton-base animate-shimmer rounded-lg {width} {height} {rounded} {className}" aria-hidden="true"></div>

<style>
  .skeleton-base {
    background: linear-gradient(90deg,
        var(--color-surface-default) 0%,
        var(--color-surface-hover) 40%,
        var(--color-surface-default) 80%);
    background-size: 600px 100%;
  }
</style>
