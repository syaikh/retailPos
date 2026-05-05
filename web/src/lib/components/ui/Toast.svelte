<script lang="ts">
  import { toast } from '$lib/stores/toast';
  import { CheckCircle, XCircle, AlertTriangle, Info, X } from 'lucide-svelte';

  const icons = {
    success: CheckCircle,
    error:   XCircle,
    warning: AlertTriangle,
    info:    Info,
  };

  const styles = {
    success: 'border-success/30 bg-success-subtle text-success-light',
    error:   'border-danger/30 bg-danger-subtle text-danger-light',
    warning: 'border-warning/30 bg-warning-subtle text-warning-light',
    info:    'border-info/30 bg-info-subtle text-info-light',
  };
</script>

<!-- Fixed toast portal -->
<div class="fixed bottom-6 right-6 z-[9999] flex flex-col gap-3 pointer-events-none">
  {#each $toast as t (t.id)}
    <div
      class="pointer-events-auto flex items-start gap-3 rounded-xl border px-4 py-3 shadow-modal
             backdrop-blur-xl min-w-72 max-w-sm animate-slide-in-right {styles[t.variant]}"
      role="alert"
    >
      <svelte:component this={icons[t.variant]} size={18} class="shrink-0 mt-0.5" />
      <p class="flex-1 text-sm font-medium leading-snug">{t.message}</p>
      <button
        onclick={() => toast.remove(t.id)}
        class="shrink-0 opacity-60 hover:opacity-100 transition-opacity -mr-1"
        aria-label="Dismiss"
      >
        <X size={14} />
      </button>
    </div>
  {/each}
</div>
