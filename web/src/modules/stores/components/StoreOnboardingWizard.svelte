<script lang="ts">
  import { Button, Input, Modal, Badge } from "$shared/ui";
  import { labels, t } from "$shared/i18n";
  import { toast } from "$shared/stores/toast.svelte";
  import { goto } from "$app/router";
  import { untrack } from "svelte";
  import { getRoles, createUser } from "$modules/admin";
  import { createStorageLocation } from "$modules/storage-location";
  import { createStoreAndGet, getReadiness } from "../services/stores-service";
  import type { Store, StoreReadiness } from "../types";
  import {
    Loader2,
    ArrowLeft,
    ArrowRight,
    Copy,
    RefreshCw,
    CheckCircle2,
    XCircle,
    Sparkles,
    Link as LinkIcon,
  } from "lucide-svelte";

  let {
    open = $bindable(false),
    onComplete = () => {},
  }: {
    open?: boolean;
    onComplete?: () => void;
  } = $props();

  type Step = "details" | "staff" | "location" | "stock" | "readiness";
  const steps: Step[] = ["details", "staff", "location", "stock", "readiness"];

  interface StaffRow {
    role: string;
    enabled: boolean;
    username: string;
    password: string;
    created: boolean;
    error: string;
  }

  let step = $state<Step>("details");
  let loading = $state(false);
  let errorMsg = $state("");
  let createdStore = $state<Store | null>(null);
  let readiness = $state<StoreReadiness | null>(null);
  let readinessLoading = $state(false);
  let copiedRole = $state("");
  let roles = $state<{ id: number; name: string }[]>([]);
  // Roles are fetched when the modal opens, so the staff step can be reached
  // before they resolve. Creating staff without them marks every row
  // "role not found" with no way forward, so the step waits for the fetch.
  let rolesReady = $state(false);
  let rolesLoad: Promise<void> | null = null;

  let storeForm = $state({ name: "", address: "", phone: "" });
  let rows = $state<StaffRow[]>([]);
  let locationForm = $state({ enabled: true, code: "", name: "" });
  // Back-navigation guards: once a step's side effect has run, replaying the
  // step must advance instead of re-POSTing (duplicate store / duplicate
  // location code) or wiping rows already marked created.
  let locationCreated = $state(false);

  const stepIndex = $derived(steps.indexOf(step));
  const currentStepLabel = $derived(stepTitle(step));

  // Address and phone are readiness blockers, so the wizard refuses to create
  // a store without them instead of surfacing the gap at the last step.
  const canSubmitDetails = $derived(
    storeForm.name.trim().length > 0 &&
      storeForm.address.trim().length > 0 &&
      storeForm.phone.trim().length > 0 &&
      !loading,
  );
  const enabledRows = $derived(rows.filter((r) => r.enabled));
  const pendingRows = $derived(enabledRows.filter((r) => !r.created));

  $effect(() => {
    if (open) {
      // Only `open` may drive the reset. The reset writes rolesReady and starts
      // the roles fetch, so tracking those reads would re-run this effect when
      // the fetch resolves: the form the user already filled is wiped and the
      // freshly reset rolesReady starts the fetch again, looping indefinitely.
      untrack(() => {
        step = "details";
        loading = false;
        errorMsg = "";
        createdStore = null;
        readiness = null;
        copiedRole = "";
        storeForm = { name: "", address: "", phone: "" };
        rows = [];
        locationForm = { enabled: true, code: "", name: "" };
        locationCreated = false;
        rolesReady = false;
        rolesLoad = null;
        void ensureRoles();
      });
    }
  });

  async function loadRoles() {
    try {
      roles = await getRoles();
      rolesReady = roles.length > 0;
    } catch {
      roles = [];
      rolesReady = false;
    }
  }

  // Dedupes concurrent callers and retries after a failed attempt: clearing the
  // in-flight handle on settle means the next staff step click fetches again.
  function ensureRoles(): Promise<void> {
    if (rolesReady) return Promise.resolve();
    if (!rolesLoad) {
      rolesLoad = loadRoles().finally(() => {
        rolesLoad = null;
      });
    }
    return rolesLoad;
  }

  function stepTitle(s: Step): string {
    switch (s) {
      case "details":
        return labels.onboardingStepDetails;
      case "staff":
        return labels.onboardingStepStaff;
      case "location":
        return labels.onboardingStepLocation;
      case "stock":
        return labels.onboardingStepStock;
      case "readiness":
        return labels.onboardingStepReadiness;
    }
  }

  function roleLabel(role: string): string {
    switch (role) {
      case "manager":
        return labels.manager;
      case "supervisor":
        return labels.supervisor;
      case "cashier":
        return labels.cashier;
      case "inventory_staff":
        return labels.inventoryStaff;
      case "finance":
        return labels.finance;
      default:
        return role;
    }
  }

  function generatePassword(): string {
    const alphabet =
      "abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789";
    const bytes = crypto.getRandomValues(new Uint8Array(16));
    return Array.from(bytes, (b) => alphabet[b % alphabet.length]).join("");
  }

  function suggestUsername(role: string, storeId: number): string {
    const base = role.replace(/[^a-z0-9]/gi, "");
    return `${base}${storeId}`;
  }

  function initStaffRows(storeId: number) {
    const required: string[] = [
      "manager",
      "supervisor",
      "cashier",
      "inventory_staff",
      "finance",
    ];
    rows = required.map((role) => {
      const username = suggestUsername(role, storeId);
      return {
        role,
        enabled: true,
        username,
        password: generatePassword(),
        created: false,
        error: "",
      };
    });
  }

  async function copyPassword(row: StaffRow) {
    try {
      await navigator.clipboard.writeText(row.password);
      copiedRole = row.role;
      setTimeout(() => (copiedRole = ""), 2000);
    } catch {
      toast.error(labels.copyFailed);
    }
  }

  function regeneratePassword(row: StaffRow) {
    row.password = generatePassword();
  }

  async function handleDetails() {
    if (!canSubmitDetails) return;
    // Reached again only via Back: the store already exists, so advancing must
    // not create a second one or reset the staff rows already marked created.
    if (createdStore) {
      step = "staff";
      return;
    }
    errorMsg = "";
    loading = true;
    const created = await createStoreAndGet({
      name: storeForm.name.trim(),
      address: storeForm.address.trim() || undefined,
      phone: storeForm.phone.trim() || undefined,
    });
    loading = false;
    if (!created) {
      errorMsg = labels.toastFailedSaveStore;
      return;
    }
    createdStore = created;
    initStaffRows(created.id);
    locationForm = {
      enabled: true,
      code: `LOC-${created.id}`.toUpperCase(),
      name: labels.defaultStorageLocation,
    };
    step = "staff";
  }

  async function handleStaff() {
    if (pendingRows.length === 0) {
      step = "location";
      return;
    }
    if (!createdStore) return;
    errorMsg = "";
    loading = true;
    // The roles fetch may still be in flight (or have failed on a previous
    // attempt) when this step is reached: resolve it before creating anyone,
    // otherwise every row would fail with "role not found".
    await ensureRoles();
    if (!rolesReady) {
      loading = false;
      errorMsg = labels.toastFailedLoadRoles;
      return;
    }
    let failed = 0;
    for (const row of pendingRows) {
      const roleRecord = roles.find((r) => r.name === row.role);
      if (!roleRecord) {
        row.error = labels.roleNotFound;
        failed++;
        continue;
      }
      try {
        await createUser({
          username: row.username.trim(),
          email: `${row.username.trim()}@example.com`,
          password: row.password,
          role_id: roleRecord.id,
          is_active: true,
          store_id: createdStore.id,
          must_change_password: true,
        });
        row.created = true;
        row.error = "";
      } catch (err) {
        row.error =
          err instanceof Error ? err.message : labels.toastFailedSaveUser;
        failed++;
      }
    }
    loading = false;
    if (failed > 0) {
      errorMsg = t("onboardingStaffFailed", { count: failed });
      return;
    }
    step = "location";
  }

  async function handleLocation() {
    if (!locationForm.enabled || !createdStore) {
      step = "stock";
      return;
    }
    // Already stored earlier in this run — replaying the step must not re-POST
    // the same code against the uniqueness constraint.
    if (locationCreated) {
      step = "stock";
      return;
    }
    if (!locationForm.code.trim() || !locationForm.name.trim()) {
      errorMsg = labels.storageLocationRequired;
      return;
    }
    errorMsg = "";
    loading = true;
    try {
      await createStorageLocation({
        code: locationForm.code.trim(),
        name: locationForm.name.trim(),
        store_id: createdStore.id,
      });
      loading = false;
      locationCreated = true;
      step = "stock";
    } catch (err) {
      loading = false;
      errorMsg =
        err instanceof Error ? err.message : labels.toastFailedSaveLocation;
    }
  }

  async function refreshReadiness() {
    if (!createdStore) return;
    readinessLoading = true;
    readiness = await getReadiness(createdStore.id);
    readinessLoading = false;
  }

  $effect(() => {
    if (step === "readiness" && createdStore) {
      void refreshReadiness();
    }
  });

  function gotoStock(path: string) {
    open = false;
    onComplete();
    goto(path);
  }

  function finish() {
    open = false;
    onComplete();
    toast.success(labels.onboardingFinished);
  }

  function blockerLabel(code: string): string {
    if (code === "store.address") return labels.blockerAddress;
    if (code === "store.phone") return labels.blockerPhone;
    if (code === "store.inactive") return labels.blockerInactive;
    if (code === "storage_location") return labels.blockerStorageLocation;
    if (code === "catalog") return labels.blockerCatalog;
    if (code.startsWith("staff.")) {
      return t("blockerStaffRole", { role: roleLabel(code.slice(6)) });
    }
    return code;
  }
</script>

<Modal
  bind:open
  title={labels.onboardingWizardTitle}
  size="lg"
  panelClass="max-h-[90vh]"
>
  <!-- Step header -->
  <div class="flex items-center justify-between gap-4 mb-5">
    <div class="flex items-center gap-2">
      {#each steps as s, i (s)}
        <span
          class="h-6 min-w-6 px-1.5 rounded-full text-xs font-semibold flex items-center justify-center {i <=
          stepIndex
            ? 'bg-primary-default text-white'
            : 'bg-surface text-text-muted'}"
        >
          {i + 1}
        </span>
        {#if i < steps.length - 1}
          <span class="w-4 h-px bg-border"></span>
        {/if}
      {/each}
    </div>
    <span class="text-xs text-text-muted">
      {t("stepOf", { current: stepIndex + 1, total: steps.length })} —
      {currentStepLabel}
    </span>
  </div>

  {#if errorMsg}
    <div
      class="mb-4 p-3 bg-danger-subtle/10 border border-danger/20 rounded-lg text-sm text-danger"
      role="alert"
    >
      {errorMsg}
    </div>
  {/if}

  {#if step === "details"}
    <div class="space-y-4">
      <p class="text-sm text-text-muted">{labels.onboardingDetailsHint}</p>
      <div>
        <label
          for="wiz-store-name"
          class="block text-sm font-medium text-text-secondary mb-2"
          >{labels.storeName} <span class="text-danger">*</span></label
        >
        <Input
          id="wiz-store-name"
          type="text"
          placeholder={labels.contohNamaToko}
          bind:value={storeForm.name}
          maxlength="100"
        />
      </div>
      <div>
        <label
          for="wiz-store-address"
          class="block text-sm font-medium text-text-secondary mb-2"
          >{labels.address}
          <span class="text-danger">*</span></label
        >
        <Input
          id="wiz-store-address"
          type="text"
          placeholder={labels.contohAlamat}
          bind:value={storeForm.address}
        />
      </div>
      <div>
        <label
          for="wiz-store-phone"
          class="block text-sm font-medium text-text-secondary mb-2"
          >{labels.phone}
          <span class="text-danger">*</span></label
        >
        <Input
          id="wiz-store-phone"
          type="text"
          placeholder={labels.contohTelepon}
          bind:value={storeForm.phone}
        />
      </div>
      <p class="text-xs text-text-muted">{labels.onboardingAddressHint}</p>
    </div>
  {:else if step === "staff"}
    <div class="space-y-4">
      <p class="text-sm text-text-muted">{labels.onboardingStaffHint}</p>
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr class="text-left text-text-muted border-b border-border">
              <th class="py-2 pr-3 font-medium w-10">{labels.include}</th>
              <th class="py-2 pr-3 font-medium">{labels.role}</th>
              <th class="py-2 pr-3 font-medium">{labels.username}</th>
              <th class="py-2 pr-3 font-medium">{labels.tempPassword}</th>
              <th class="py-2 font-medium w-24">{labels.actions}</th>
            </tr>
          </thead>
          <tbody>
            {#each rows as row (row.role)}
              <tr class="border-b border-border/50 align-top">
                <td class="py-2 pr-3">
                  <input
                    type="checkbox"
                    class="w-4 h-4 rounded accent-[var(--color-primary-default,#2563eb))]"
                    bind:checked={row.enabled}
                    aria-label={t("includeRole", { role: roleLabel(row.role) })}
                    disabled={row.created}
                  />
                </td>
                <td class="py-2 pr-3">
                  <span class="font-medium">{roleLabel(row.role)}</span>
                  {#if row.created}
                    <span class="ml-2 text-success">
                      <CheckCircle2 size={14} />
                    </span>
                  {/if}
                </td>
                <td class="py-2 pr-3">
                  <Input
                    type="text"
                    bind:value={row.username}
                    disabled={row.created || !row.enabled}
                    placeholder={labels.username}
                  />
                </td>
                <td class="py-2 pr-3">
                  <div class="flex items-center gap-1">
                    <Input
                      type="text"
                      bind:value={row.password}
                      disabled={row.created || !row.enabled}
                      class="font-mono text-xs"
                    />
                    <Button
                      variant="ghost"
                      size="icon"
                      title={labels.regeneratePassword}
                      aria-label={labels.regeneratePassword}
                      onclick={() => regeneratePassword(row)}
                      disabled={row.created || !row.enabled}
                    >
                      <RefreshCw size={14} />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      title={copiedRole === row.role
                        ? labels.copied
                        : labels.copy}
                      aria-label={labels.copy}
                      onclick={() => copyPassword(row)}
                      disabled={!row.enabled}
                    >
                      {#if copiedRole === row.role}
                        <CheckCircle2 size={14} class="text-success" />
                      {:else}
                        <Copy size={14} />
                      {/if}
                    </Button>
                  </div>
                  {#if row.error}
                    <p class="text-xs text-danger mt-1" role="alert">
                      {row.error}
                    </p>
                  {/if}
                </td>
                <td class="py-2 text-xs text-text-muted">
                  {row.enabled
                    ? row.created
                      ? labels.created
                      : labels.willCreate
                    : labels.skipped}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
      <div class="p-3 bg-warning-subtle/10 rounded-lg flex items-start gap-2">
        <Sparkles size={16} class="text-warning shrink-0 mt-0.5" />
        <p class="text-sm text-text-secondary">
          {labels.onboardingPasswordNote}
        </p>
      </div>
    </div>
  {:else if step === "location"}
    <div class="space-y-4">
      <p class="text-sm text-text-muted">{labels.onboardingLocationHint}</p>
      <label class="flex items-center gap-2 text-sm text-text-secondary">
        <input
          type="checkbox"
          class="w-4 h-4"
          bind:checked={locationForm.enabled}
        />
        {labels.createStorageLocation}
      </label>
      {#if locationForm.enabled}
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label
              for="wiz-loc-code"
              class="block text-sm font-medium text-text-secondary mb-2"
              >{labels.storageLocationCode}
              <span class="text-danger">*</span></label
            >
            <Input
              id="wiz-loc-code"
              type="text"
              bind:value={locationForm.code}
            />
          </div>
          <div>
            <label
              for="wiz-loc-name"
              class="block text-sm font-medium text-text-secondary mb-2"
              >{labels.storageLocationName}
              <span class="text-danger">*</span></label
            >
            <Input
              id="wiz-loc-name"
              type="text"
              bind:value={locationForm.name}
            />
          </div>
        </div>
      {/if}
    </div>
  {:else if step === "stock"}
    <div class="space-y-4">
      <p class="text-sm text-text-muted">{labels.onboardingStockHint}</p>
      <div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
        <button
          type="button"
          onclick={() => gotoStock("/inventory/products")}
          class="p-4 rounded-xl border border-border bg-surface text-left hover:border-primary transition-colors"
        >
          <LinkIcon size={16} class="text-primary mb-2" />
          <p class="text-sm font-semibold text-text-primary">
            {labels.stockImportCsv}
          </p>
          <p class="text-xs text-text-muted mt-1">
            {labels.stockImportCsvDesc}
          </p>
        </button>
        <button
          type="button"
          onclick={() => gotoStock("/purchase-orders")}
          class="p-4 rounded-xl border border-border bg-surface text-left hover:border-primary transition-colors"
        >
          <LinkIcon size={16} class="text-primary mb-2" />
          <p class="text-sm font-semibold text-text-primary">
            {labels.stockPurchaseOrder}
          </p>
          <p class="text-xs text-text-muted mt-1">
            {labels.stockPurchaseOrderDesc}
          </p>
        </button>
        <button
          type="button"
          onclick={() => gotoStock("/stock-opnames/adjustments")}
          class="p-4 rounded-xl border border-border bg-surface text-left hover:border-primary transition-colors"
        >
          <LinkIcon size={16} class="text-primary mb-2" />
          <p class="text-sm font-semibold text-text-primary">
            {labels.stockAdjustStock}
          </p>
          <p class="text-xs text-text-muted mt-1">
            {labels.stockAdjustStockDesc}
          </p>
        </button>
      </div>
    </div>
  {:else if step === "readiness"}
    <div class="space-y-4">
      {#if readinessLoading}
        <div class="flex items-center gap-2 text-text-muted text-sm">
          <Loader2 size={16} class="animate-spin" />
          {labels.loading}
        </div>
      {:else if !readiness}
        <p class="text-sm text-text-muted">{labels.readinessUnavailable}</p>
      {:else}
        <div class="flex items-center gap-3">
          <Badge variant={readiness.ready ? "success" : "warning"}>
            {readiness.ready ? labels.ready : labels.notReady}
          </Badge>
          <span class="text-sm text-text-muted">{readiness.name}</span>
          <button
            type="button"
            class="text-xs text-primary hover:underline"
            onclick={refreshReadiness}
          >
            {labels.refresh}
          </button>
        </div>

        <ul class="space-y-2 text-sm">
          <li class="flex items-center gap-2">
            {#if readiness.address_set && readiness.phone_set}
              <CheckCircle2 size={16} class="text-success" />
            {:else}
              <XCircle size={16} class="text-danger" />
            {/if}
            {labels.onboardingCheckProfile}
          </li>
          {#each readiness.required_roles as role (role)}
            <li class="flex items-center gap-2">
              {#if readiness.staff[role]}
                <CheckCircle2 size={16} class="text-success" />
              {:else}
                <XCircle size={16} class="text-danger" />
              {/if}
              {t("onboardingRoleStaffed", {
                role: roleLabel(role),
                count: readiness.staff[role] ?? 0,
              })}
            </li>
          {/each}
          <li class="flex items-center gap-2">
            {#if readiness.storage_locations > 0}
              <CheckCircle2 size={16} class="text-success" />
            {:else}
              <XCircle size={16} class="text-danger" />
            {/if}
            {t("onboardingLocations", { count: readiness.storage_locations })}
          </li>
          <li class="flex items-center gap-2">
            {#if readiness.catalog.active_products > 0 && readiness.catalog.zero_stock_products < readiness.catalog.active_products}
              <CheckCircle2 size={16} class="text-success" />
            {:else}
              <XCircle size={16} class="text-danger" />
            {/if}
            {t("onboardingCatalog", {
              active: readiness.catalog.active_products,
              zero: readiness.catalog.zero_stock_products,
            })}
          </li>
        </ul>

        {#if readiness.blockers.length > 0}
          <div class="p-3 bg-danger-subtle/10 rounded-lg">
            <p class="text-sm font-semibold text-danger mb-1">
              {labels.blockersRemaining}
            </p>
            <ul class="text-sm text-danger space-y-0.5">
              {#each readiness.blockers as code (code)}
                <li>— {blockerLabel(code)}</li>
              {/each}
            </ul>
          </div>
        {/if}

        <div class="flex flex-wrap gap-2 pt-2">
          <Button variant="secondary" onclick={() => goto("/pos")}>
            {labels.goToPos}
          </Button>
          <Button variant="secondary" onclick={() => goto("/admin/users")}>
            {labels.goToUsers}
          </Button>
        </div>
      {/if}
    </div>
  {/if}

  {#snippet footer()}
    <div class="flex items-center gap-3 w-full">
      {#if step !== "details"}
        <Button
          variant="secondary"
          onclick={() => {
            const i = steps.indexOf(step);
            if (i > 0) step = steps[i - 1];
            errorMsg = "";
          }}
        >
          <ArrowLeft size={14} />
          {labels.back}
        </Button>
      {:else}
        <Button variant="secondary" onclick={() => (open = false)}>
          {labels.cancel}
        </Button>
      {/if}
      <div class="flex-1"></div>
      {#if step === "details"}
        <Button
          variant="primary"
          disabled={!canSubmitDetails}
          onclick={handleDetails}
        >
          {#if loading}
            <Loader2 size={16} class="animate-spin" /> {labels.saving}
          {:else}
            {labels.next} <ArrowRight size={14} />
          {/if}
        </Button>
      {:else if step === "staff"}
        <Button variant="secondary" onclick={() => (step = "location")}>
          {labels.skip}
        </Button>
        <Button
          variant="primary"
          disabled={loading || !rolesReady}
          onclick={handleStaff}
        >
          {#if loading}
            <Loader2 size={16} class="animate-spin" /> {labels.saving}
          {:else}
            {pendingRows.length > 0
              ? t("createStaffCount", { count: pendingRows.length })
              : labels.next}
            <ArrowRight size={14} />
          {/if}
        </Button>
      {:else if step === "location"}
        <Button variant="secondary" onclick={() => (step = "stock")}>
          {labels.skip}
        </Button>
        <Button variant="primary" disabled={loading} onclick={handleLocation}>
          {#if loading}
            <Loader2 size={16} class="animate-spin" /> {labels.saving}
          {:else}
            {locationForm.enabled ? labels.save : labels.next}
            <ArrowRight size={14} />
          {/if}
        </Button>
      {:else if step === "stock"}
        <Button variant="primary" onclick={() => (step = "readiness")}>
          {labels.next}
          <ArrowRight size={14} />
        </Button>
      {:else}
        <Button variant="primary" onclick={finish}>{labels.finish}</Button>
      {/if}
    </div>
  {/snippet}
</Modal>
