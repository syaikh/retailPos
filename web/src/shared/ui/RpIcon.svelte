<script lang="ts">
  let { size = 28, class: className = "" } = $props();
</script>

<span 
  class={className} 
  style="font-size: {size * 0.7}px; font-weight: 700; line-height: 1; width: {size}px; height: {size}px; display: flex; align-items: center; justify-content: center;"
>
  Rp
</span>
