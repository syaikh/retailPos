<script lang="ts">
  let {
    action = '',
    class: className = '',
  }: {
    action: string;
    class?: string;
  } = $props();

  let badgeClass = $derived(() => {
    const v = (action || '').toUpperCase();
    if (v === 'CREATE') return 'bg-success-subtle text-success-light border-success-default/20';
    if (v === 'UPDATE') return 'bg-warning-subtle text-warning-light border-warning-default/20';
    if (v === 'DELETE') return 'bg-danger-subtle text-danger-light border-danger-default/20';
    if (v === 'LOGIN') return 'bg-primary-subtle text-primary-light border-primary-default/20';
    if (v === 'LOGOUT') return 'bg-surface-hover text-text-secondary border-border-strong';
    return 'bg-surface-default text-text-muted border-border';
  });
</script>

<span
  class="inline-flex items-center justify-center px-2.5 h-6 rounded-full text-sm font-medium border leading-none {badgeClass()} {className}"
>
  {action || '—'}
</span>
