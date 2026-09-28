<script lang="ts">
  import { Button } from "$shared/ui";
  import { labels } from "$shared/i18n";
  import { logout, useAuthStore } from "$modules/auth";
  import { fade, fly } from "svelte/transition";
  import { LockKeyhole } from "lucide-svelte";
  import PasswordChangeForm from "./PasswordChangeForm.svelte";

  const auth = useAuthStore();

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

      <PasswordChangeForm idPrefix="force" autofocus={auth.mustChangePassword}>
        {#snippet actions()}
          <Button variant="ghost" type="button" onclick={handleLogout}>
            {labels.signOut}
          </Button>
        {/snippet}
      </PasswordChangeForm>
    </div>
  </div>
{/if}
