<script lang="ts">
  import { cn } from '$shared/utils/cn';

  let {
    value = $bindable(),
    placeholder = '0',
    class: className = '',
    disabled = false,
    required = false,
    id = '',
  }: {
    value?: number;
    placeholder?: string;
    class?: string;
    disabled?: boolean;
    required?: boolean;
    id?: string;
  } = $props();

  let el: HTMLInputElement | undefined = $state();

  function fmt(n: number): string {
    return n ? n.toLocaleString('id-ID') : '';
  }

  function cursorPos(rawAfter: string, pos: number): number {
    let s = fmt(parseInt(rawAfter, 10) || 0);
    let ri = 0;
    for (let fi = 0; fi < s.length; fi++) {
      if (s[fi] === '.') continue;
      if (ri >= pos) return fi;
      ri++;
    }
    return s.length;
  }

  function handleInput() {
    if (!el) return;
    const sel = el.selectionStart ?? 0;
    const dots = (el.value.slice(0, sel).match(/\./g) || []).length;
    const rc = sel - dots;

    const raw = el.value.replace(/[^0-9]/g, '');
    const nv = raw ? parseInt(raw, 10) : 0;
    value = nv;

    const formatted = fmt(nv);
    if (el.value !== formatted) {
      const nc = cursorPos(raw, rc);
      el.value = formatted;
      el.setSelectionRange(nc, nc);
    }
  }

  $effect(() => {
    if (el) {
      const formatted = fmt(value);
      if (el.value !== formatted) {
        el.value = formatted;
      }
    }
  });
</script>

<div class={cn(
  'flex items-center gap-1.5 bg-bg-secondary border border-border-default rounded-xl px-3 h-[42px] w-full transition-colors duration-200',
  value > 0 ? 'border-primary-default' : '',
  disabled ? 'opacity-40 cursor-not-allowed' : '',
  className
)}>
  <span class="text-xs text-text-muted font-medium shrink-0 select-none">Rp</span>
  <input
    {id}
    bind:this={el}
    type="text"
    inputmode="numeric"
    {placeholder}
    {disabled}
    {required}
    class="w-full bg-transparent text-sm text-right text-text-primary outline-none placeholder:text-text-muted disabled:cursor-not-allowed"
    oninput={handleInput}
  />
</div>
