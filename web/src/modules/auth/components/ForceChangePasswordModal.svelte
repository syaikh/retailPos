<script lang="ts">
  import { Button, Input } from "$shared/ui";
  import { labels } from "$shared/i18n";
  import { toast } from "$shared/stores/toast.svelte";
  import { changePassword, logout, useAuthStore } from "$modules/auth";
  import { fade, fly } from "svelte/transition";
  import { LockKeyhole } from "lucide-svelte";

  const auth = useAuthStore();

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
    if (auth.mustChangePassword) {
      fieldRef?.focus();
    }
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

  async function handleLogout() {
    await logout();
  }
</script>

{#if auth.mustChangePassword}
  <div
    class="fixed inset-0 z-[100] flex items-center justify-center p-4 bg-black/70"
    transition:fade={{ duration: 200 }}
    role="presentation"
  >
    <div
      class="w-full max-w-md bg-surface-default border border-border rounded-2xl shadow-modal p-6 flex flex-col gap-4"
      transition:fly={{ y: 20, duration: 300 }}
      role="dialog"
      aria-modal="true"
      aria-labelledby="force-change-password-title"
    >
      <div class="flex items-center gap-3">
        <div
          class="w-10 h-10 rounded-xl bg-primary-default/15 text-primary-default flex items-center justify-center"
        >
          <LockKeyhole size={20} />
        </div>
        <div>
          <h2
            id="force-change-password-title"
            class="text-base font-semibold text-text-primary"
          >
            {labels.forceChangePasswordTitle}
          </h2>
          <p class="text-xs text-text-muted">
            {labels.forceChangePasswordDesc}
          </p>
        </div>
      </div>

      <form class="flex flex-col gap-3" onsubmit={handleSubmit}>
        <div>
          <label
            for="force-current-password"
            class="block text-sm font-medium text-text-secondary mb-1.5"
            >{labels.currentPassword}</label
          >
          <Input
            id="force-current-password"
            type="password"
            autocomplete="current-password"
            bind:value={currentPassword}
            elementRef={(el) => (fieldRef = el as HTMLInputElement)}
          />
        </div>
        <div>
          <label
            for="force-new-password"
            class="block text-sm font-medium text-text-secondary mb-1.5"
            >{labels.newPassword}</label
          >
          <Input
            id="force-new-password"
            type="password"
            autocomplete="new-password"
            bind:value={newPassword}
            error={tooShort ? labels.passwordMinLength : ""}
          />
        </div>
        <div>
          <label
            for="force-confirm-password"
            class="block text-sm font-medium text-text-secondary mb-1.5"
            >{labels.confirmNewPassword}</label
          >
          <Input
            id="force-confirm-password"
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
          <Button variant="ghost" type="button" onclick={handleLogout}>
            {labels.signOut}
          </Button>
          <Button type="submit" disabled={!canSubmit}>
            {loading ? labels.saving : labels.save}
          </Button>
        </div>
      </form>
    </div>
  </div>
{/if}
