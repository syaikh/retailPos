<script lang="ts">
  let {
    checked = $bindable(false),
    label = '',
    disabled = false,
    id,
  }: {
    checked?: boolean;
    label?: string;
    disabled?: boolean;
    id?: string;
  } = $props();
</script>

<label class="flex items-center gap-3 cursor-pointer select-none group" for={id}>
  <div class="relative">
    <input
      {id}
      type="checkbox"
      class="sr-only peer"
      bind:checked
      {disabled}
    />
    <div class="w-10 h-5 bg-surface-default border border-border rounded-full peer peer-checked:bg-primary-subtle peer-checked:border-primary/50 transition-colors {disabled ? 'opacity-50' : ''}"></div>
    <div class="absolute left-1 top-1 w-3 h-3 bg-text-muted rounded-full peer-checked:translate-x-5 peer-checked:bg-primary-light transition-transform shadow-sm"></div>
  </div>
  {#if label}
    <span class="text-sm font-medium text-text-secondary group-hover:text-text-primary transition-colors">
      {label}
    </span>
  {/if}
</label>
