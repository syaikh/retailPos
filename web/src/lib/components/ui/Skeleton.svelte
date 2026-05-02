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

<div class="skeleton {width} {height} {rounded} {className}" aria-hidden="true"></div>
