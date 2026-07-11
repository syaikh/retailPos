<script lang="ts">
  import { Button } from '$shared/ui';
  import { Home, ArrowLeft } from 'lucide-svelte';
  import { goto } from '$app/router';
</script>

<div class="min-h-[60vh] flex items-center justify-center">
  <div class="text-center">
    <div class="w-20 h-20 mx-auto mb-6 rounded-2xl bg-surface-default border border-border flex items-center justify-center">
      <span class="text-4xl font-bold text-text-muted">404</span>
    </div>
    <h1 class="text-2xl font-bold text-text-primary mb-2">Page Not Found</h1>
    <p class="text-text-muted text-sm mb-6 max-w-sm mx-auto">
      The page you're looking for doesn't exist or has been moved.
    </p>
    <div class="flex items-center justify-center gap-3">
      <Button variant="secondary" onclick={() => window.history.back()}>
        <ArrowLeft size={16} class="mr-1" /> Go Back
      </Button>
      <Button variant="primary" onclick={() => goto('/')}>
        <Home size={16} class="mr-1" /> Dashboard
      </Button>
    </div>
  </div>
</div>
