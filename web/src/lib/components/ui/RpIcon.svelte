<script lang="ts">
  let { size = 20, class: className = "" } = $props();
</script>

<svg
  xmlns="http://www.w3.org/2000/svg"
  width={size}
  height={size}
  viewBox="0 0 24 24"
  fill="none"
  stroke="currentColor"
  stroke-width="2"
  stroke-linecap="round"
  stroke-linejoin="round"
  class={className}
>
  <path d="M4 6h4a2 2 0 0 1 0 4H4v10" />
  <path d="M4 10h4" />
  <path d="M12 12h3a2 2 0 0 1 0 4h-3v4" />
  <path d="M12 16h3" />
</svg>
