<script lang="ts">
  let {
    level = 'aman',
    value = 0
  }: {
    level?: 'rendah' | 'sedang' | 'aman';
    value?: number;
  } = $props();
</script>

<span class="stock-badge {level}">
  {value} pcs
</span>

<style>
  .stock-badge {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    padding: 4px 10px;
    border-radius: 9999px;
    font-size: 0.8rem;
    font-weight: 600;
    min-width: 45px;
  }

  .rendah {
    background: rgba(239, 68, 68, 0.15);
    color: #ef4444;
  }

  .sedang {
    background: rgba(245, 158, 11, 0.15);
    color: #f59e0b;
  }

  .aman {
    background: rgba(16, 185, 129, 0.15);
    color: #10b981;
  }
</style>