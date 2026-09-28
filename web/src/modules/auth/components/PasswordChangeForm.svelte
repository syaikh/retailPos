<script lang="ts">
  import type { Snippet } from "svelte";
  import { Button, Input } from "$shared/ui";
  import { labels } from "$shared/i18n";
  import { toast } from "$shared/stores/toast.svelte";
  import { changePassword } from "$modules/auth";

  let {
    idPrefix = "account",
    autofocus = false,
    actions,
  }: {
    idPrefix?: string;
    autofocus?: boolean;
    actions?: Snippet;
  } = $props();

  let currentPassword = $state("");
  let newPassword = $state("");
  let confirmPassword = $state("");
  let errorMsg = $state("");
  let loading = $state(false);
  let fieldRef = $state<HTMLInputElement>();

  const tooShort = $derived(newPassword.length > 0 && newPassword.length < 8);
  const mismatch = $derived(
    confirmPassword.length > 0 && newPassword !== confirmPassword,
  );
  const canSubmit = $derived(
    currentPassword.length > 0 &&
      newPassword.length >= 8 &&
      newPassword === confirmPassword &&
      !loading,
  );

  $effect(() => {
    if (autofocus) fieldRef?.focus();
  });

  async function handleSubmit(e: Event) {
    e.preventDefault();
    if (!canSubmit) return;
    errorMsg = "";
    loading = true;
    const result = await changePassword(currentPassword, newPassword);
    loading = false;
    if (result.ok) {
      toast.success(labels.passwordChangedSuccessfully);
      currentPassword = "";
      newPassword = "";
      confirmPassword = "";
    } else {
      errorMsg = result.message;
    }
  }
</script>

<form class="flex flex-col gap-3" onsubmit={handleSubmit}>
  <div>
    <label
      for={`${idPrefix}-current-password`}
      class="block text-sm font-medium text-text-secondary mb-1.5"
      >{labels.currentPassword}</label
    >
    <Input
      id={`${idPrefix}-current-password`}
      type="password"
      autocomplete="current-password"
      bind:value={currentPassword}
      elementRef={(el) => (fieldRef = el as HTMLInputElement)}
    />
  </div>
  <div>
    <label
      for={`${idPrefix}-new-password`}
      class="block text-sm font-medium text-text-secondary mb-1.5"
      >{labels.newPassword}</label
    >
    <Input
      id={`${idPrefix}-new-password`}
      type="password"
      autocomplete="new-password"
      bind:value={newPassword}
      error={tooShort ? labels.passwordMinLength : ""}
    />
  </div>
  <div>
    <label
      for={`${idPrefix}-confirm-password`}
      class="block text-sm font-medium text-text-secondary mb-1.5"
      >{labels.confirmNewPassword}</label
    >
    <Input
      id={`${idPrefix}-confirm-password`}
      type="password"
      autocomplete="new-password"
      bind:value={confirmPassword}
      error={mismatch ? labels.passwordsDoNotMatch : ""}
    />
  </div>

  {#if errorMsg}
    <p class="text-sm text-danger" role="alert">{errorMsg}</p>
  {/if}

  <div class="flex items-center justify-between gap-3 pt-2">
    <div class="flex items-center gap-2">
      {#if actions}
        {@render actions()}
      {/if}
    </div>
    <Button type="submit" disabled={!canSubmit}>
      {loading ? labels.saving : labels.save}
    </Button>
  </div>
</form>
