<script lang="ts">
  import { cn } from '$shared/utils/cn';

  let {
    value = $bindable(0),
    class: className = '',
    placeholder = '',
    disabled = false,
    id,
  }: {
    value?: number;
    class?: string;
    placeholder?: string;
    disabled?: boolean;
    id?: string;
  } = $props();

  let focused = $state(false);
  let rawInput = $state('');
  let inputEl: HTMLInputElement;

  function formatDisplay(n: number): string {
    return n.toLocaleString('id-ID');
  }

  function parseInput(s: string): number {
    const cleaned = s.replace(/[^0-9]/g, '');
    return cleaned === '' ? 0 : Number(cleaned);
  }

  function handleFocus() {
    focused = true;
    rawInput = String(value ?? 0);
    setTimeout(() => inputEl?.select(), 0);
  }

  function handleBlur() {
    focused = false;
    const parsed = parseInput(rawInput);
    value = parsed;
    rawInput = formatDisplay(parsed);
  }

  function handleInput(e: Event) {
    const v = (e.target as HTMLInputElement).value;
    rawInput = v.replace(/[^0-9]/g, '');
    const parsed = parseInput(rawInput);
    value = parsed;
  }

  $effect(() => {
    if (!focused) {
      rawInput = formatDisplay(value ?? 0);
    }
  });
</script>

<input
  bind:this={inputEl}
  id={id ?? crypto.randomUUID()}
  type="text"
  inputmode="numeric"
  {placeholder}
  {disabled}
  value={focused ? rawInput : formatDisplay(value ?? 0)}
  onfocus={handleFocus}
  onblur={handleBlur}
  oninput={handleInput}
  class={cn(
    'w-full rounded-xl border bg-bg-secondary px-3.5 py-2.5 text-sm text-text-primary placeholder-text-muted focus:outline-none focus:ring-2 disabled:cursor-not-allowed disabled:opacity-40 transition-colors duration-200',
    'border-border-default focus:border-primary-default focus:ring-primary-default/20',
    className
  )}
/>
