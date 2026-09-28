<script lang="ts">
  import { PageHeader, Card } from "$shared/ui";
  import { labels } from "$shared/i18n";
  import PasswordChangeForm from "$modules/auth/components/PasswordChangeForm.svelte";
</script>

<div class="max-w-4xl mx-auto px-4 sm:px-6 py-8 space-y-8">
  <PageHeader
    title={labels.changePassword}
    subtitle={labels.changePasswordDesc}
  />
  <Card class="max-w-xl p-6">
    <PasswordChangeForm autofocus />
  </Card>
</div>
