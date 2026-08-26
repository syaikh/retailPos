<script lang="ts">
  import { printConfig } from '$shared/stores/printConfig.svelte';
  import { labels } from '$shared/i18n';

  let editing = $state(false);
  let urlDraft = $state(printConfig.agentUrl);
  let status = $state<'idle' | 'ok' | 'err'>('idle');

  function openEditor() {
    urlDraft = printConfig.agentUrl;
    editing = !editing;
    status = 'idle';
  }

  function saveUrl() {
    printConfig.setAgentUrl(urlDraft);
    editing = false;
  }

  async function testConnection() {
    status = 'idle';
    try {
      const base = printConfig.agentUrl.replace(/\/+$/, '');
      const res = await fetch(`${base}/health`);
      status = res.ok ? 'ok' : 'err';
    } catch {
      status = 'err';
    }
  }

  const segBase = 'px-2 py-0.5 text-[11px] font-medium rounded transition-colors';
</script>

<div class="flex items-center justify-between gap-2 px-1">
  <span class="text-text-muted text-[11px]">{labels.print}</span>
  <div class="flex items-center rounded-md border border-border overflow-hidden">
    <button
      type="button"
      class="{segBase} {printConfig.mode === 'preview' ? 'bg-primary-subtle text-primary-light' : 'text-text-muted hover:text-text-secondary'}"
      onclick={() => printConfig.setMode('preview')}
      title="Show 58mm preview and browser print dialog"
    >{labels.preview}</button>
    <button
      type="button"
      class="{segBase} {printConfig.mode === 'silent' ? 'bg-primary-subtle text-primary-light' : 'text-text-muted hover:text-text-secondary'}"
      onclick={() => printConfig.setMode('silent')}
      title="Send silently to the local print agent (no dialog)"
    >{labels.silent}</button>
  </div>
  <button
    type="button"
    class="text-text-muted hover:text-text-secondary text-[11px] px-1"
    onclick={openEditor}
    title="Print agent settings"
  >⚙</button>
</div>

{#if editing}
  <div class="flex items-center gap-1 px-1 pb-1">
    <input
      class="flex-1 min-w-0 rounded border border-border bg-surface-default px-1.5 py-0.5 text-[11px] text-text-primary"
      bind:value={urlDraft}
      placeholder="http://localhost:9123"
      spellcheck="false"
    />
    <button type="button" class="text-[11px] px-1.5 py-0.5 rounded bg-primary-subtle text-primary-light" onclick={saveUrl}>{labels.save}</button>
    <button type="button" class="text-[11px] px-1.5 py-0.5 rounded border border-border text-text-muted" onclick={testConnection}>Test</button>
  </div>
  {#if status === 'ok'}
    <p class="px-1 text-[10px] text-emerald-600">● {labels.agentConnected}</p>
  {:else if status === 'err'}
    <p class="px-1 text-[10px] text-rose-600">● {labels.agentUnreachable}</p>
  {/if}
{/if}
