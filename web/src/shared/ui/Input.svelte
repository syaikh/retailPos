<script lang="ts">
  import { cn } from '$shared/utils/cn';
  import type { Snippet } from 'svelte';

  let {
    tag = 'input',
    class: className = '',
    value = $bindable(),
    children,
    oninput: externalOninput,
    elementRef,
    error = '',
    selectOnFocus = false,
    ...rest
  }: {
    tag?: 'input' | 'select' | 'textarea';
    class?: string;
    value?: string | number;
    children?: Snippet;
    oninput?: (e: Event) => void;
    elementRef?: (el: HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement) => void;
    error?: string;
    selectOnFocus?: boolean;
    [key: string]: unknown;
  } = $props();

  const inputId = crypto.randomUUID();

  function handleInput(e: Event) {
    const target = e.target as HTMLInputElement | HTMLSelectElement;
    const rawValue =
      target.tagName === 'SELECT'
        ? ((target as HTMLSelectElement).selectedOptions?.[0] as
            | (HTMLOptionElement & { __value?: unknown })
            | undefined)?.__value ?? (target as HTMLSelectElement).value
        : (target as HTMLInputElement).value;
    value = rest.type === 'number' ? (rawValue === '' ? 0 : Number(rawValue)) : rawValue;
    externalOninput?.(e);
  }

  function refAction(el: Element) {
    elementRef?.(el as HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement);
    return {};
  }

  function handleFocus(e: FocusEvent) {
    if (selectOnFocus) {
      (e.target as HTMLInputElement | HTMLTextAreaElement).select();
    }
  }
</script>

{#if tag === 'input'}
  <svelte:element
    this={tag}
    class={cn(
      'w-full rounded-xl border bg-bg-secondary px-3.5 py-2.5 text-sm text-text-primary placeholder-text-muted focus:outline-none focus:ring-2 disabled:cursor-not-allowed disabled:opacity-40 transition-colors duration-200',
      error ? 'border-danger focus:border-danger focus:ring-danger/20' : 'border-border-default focus:border-primary-default focus:ring-primary-default/20',
      className
    )}
    id={inputId}
    {value}
    aria-invalid={!!error}
    aria-describedby={error ? `${inputId}-error` : undefined}
    oninput={handleInput}
    onfocus={handleFocus}
    use:refAction
    {...rest}
  />
  {#if error}
    <p id="{inputId}-error" class="text-xs text-danger mt-1" role="alert">{error}</p>
  {/if}
{:else}
  <svelte:element
    this={tag}
    class={cn(
      'w-full rounded-xl border bg-bg-secondary px-3.5 py-2.5 text-sm text-text-primary placeholder-text-muted focus:outline-none focus:ring-2 disabled:cursor-not-allowed disabled:opacity-40 transition-colors duration-200',
      error ? 'border-danger focus:border-danger focus:ring-danger/20' : 'border-border-default focus:border-primary-default focus:ring-primary-default/20',
      className
    )}
    id={inputId}
    {value}
    aria-invalid={!!error}
    aria-describedby={error ? `${inputId}-error` : undefined}
    oninput={handleInput}
    onfocus={handleFocus}
    use:refAction
    {...rest}
  >
    {#if tag === 'select'}
      {@render children?.()}
    {/if}
  </svelte:element>
  {#if error}
    <p id="{inputId}-error" class="text-xs text-danger mt-1" role="alert">{error}</p>
  {/if}
{/if}
