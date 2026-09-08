<script lang="ts">
  import Input from './Input.svelte';

  let {
    value = $bindable(0),
    class: className = '',
    ...rest
  }: {
    value?: number;
    class?: string;
    [key: string]: unknown;
  } = $props();
</script>

<Input type="number" bind:value selectOnFocus class={className} {...rest} />
