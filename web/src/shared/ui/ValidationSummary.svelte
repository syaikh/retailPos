<script lang="ts">
  import { AlertCircle, Info } from 'lucide-svelte';
  import type { ValidationError } from '$shared/types/import-export';

  let {
    errors = [] as ValidationError[],
    warnings = [] as ValidationError[],
  }: {
    errors?: ValidationError[];
    warnings?: ValidationError[];
  } = $props();

  let hasContent = $derived(errors.length > 0 || warnings.length > 0);
</script>

{#if hasContent}
  <div class="space-y-2">
    {#if errors.length > 0}
      <div class="p-3 bg-danger-subtle/10 rounded-lg">
        <p class="text-xs font-semibold text-danger mb-2 flex items-center gap-1.5">
          <AlertCircle size={12} />
          {errors.length} error{errors.length !== 1 ? 's' : ''}
        </p>
        <div class="max-h-48 overflow-y-auto space-y-1">
          {#each errors as err}
            <p class="text-xs text-danger/80 flex items-start gap-1.5">
              <span class="shrink-0 mt-0.5">•</span>
              <span>
                Row {err.row}: <strong>{err.field}</strong> — {err.reason}
                {#if err.suggestion}
                  <span class="text-text-muted">({err.suggestion})</span>
                {/if}
              </span>
            </p>
          {/each}
        </div>
      </div>
    {/if}

    {#if warnings.length > 0}
      <div class="p-3 bg-amber-500/10 rounded-lg">
        <p class="text-xs font-semibold text-warning-light mb-2 flex items-center gap-1.5">
          <Info size={12} />
          {warnings.length} warning{warnings.length !== 1 ? 's' : ''}
        </p>
        <div class="max-h-32 overflow-y-auto space-y-1">
          {#each warnings as warn}
            <p class="text-xs text-warning-light/80 flex items-start gap-1.5">
              <span class="shrink-0 mt-0.5">•</span>
              <span>
                Row {warn.row}: <strong>{warn.field}</strong> — {warn.reason}
              </span>
            </p>
          {/each}
        </div>
      </div>
    {/if}
  </div>
{/if}
