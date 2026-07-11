<script lang="ts">
  import { Skeleton } from '$shared/ui';

  let {
    rows = 5,
    variant = 'table',
  }: {
    rows?: number;
    variant?: 'table' | 'list';
  } = $props();
</script>

{#if variant === 'list'}
  <div class="divide-y divide-border">
    {#each Array(rows) as _}
      <div class="flex items-center gap-4 px-4 py-3.5">
        <Skeleton width="w-32" height="h-4" />
        <Skeleton width="w-24" height="h-4" />
        <Skeleton width="w-20" height="h-6" rounded="rounded-full" class="ml-auto" />
        <Skeleton width="w-28" height="h-4" />
      </div>
    {/each}
  </div>
{:else}
  <div class="divide-y divide-border">
    {#each Array(rows) as _}
      <div class="flex items-center gap-4 px-4 py-3">
        <Skeleton width="w-4" height="h-4" />
        <Skeleton width="w-1/3" height="h-4" />
        <Skeleton width="w-1/4" height="h-4" />
        <Skeleton width="w-20" height="h-6" rounded="rounded-full" />
        <Skeleton width="w-1/5" height="h-4" class="ml-auto" />
      </div>
    {/each}
  </div>
{/if}
