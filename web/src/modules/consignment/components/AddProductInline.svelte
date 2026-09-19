<script lang="ts">
  import { Button, Input, Modal, NumberInput } from "$shared/ui";
  import { toast } from "$shared/stores/toast.svelte";
  import { getApiErrorMessage } from "$shared/utils/error-utils";
  import { labels } from "$shared/i18n";
  import {
    createProduct,
    getNextSku,
  } from "$modules/product/services/product-service";

  let {
    open = $bindable(false),
    oncreated,
  }: {
    open: boolean;
    oncreated?: (product: {
      id: number;
      sku: string;
      name: string;
      price: number;
    }) => void;
  } = $props();

  let saving = $state(false);
  let skuLoading = $state(false);
  let form = $state({
    sku: "",
    name: "",
    price: 0,
  });

  async function loadSku() {
    skuLoading = true;
    try {
      form.sku = await getNextSku();
    } catch {
      // User can enter manually
    } finally {
      skuLoading = false;
    }
  }

  async function submit() {
    if (!form.sku.trim()) {
      toast.error(labels.consignmentSkuRequired);
      return;
    }
    if (!form.name.trim()) {
      toast.error(labels.errorNameRequired);
      return;
    }
    if (form.price < 0) {
      toast.error(labels.consignmentPriceNotNegative);
      return;
    }

    saving = true;
    try {
      const product = await createProduct({
        sku: form.sku.trim(),
        name: form.name.trim(),
        barcode: "",
        category: "",
        brand_id: null,
        price: form.price,
        cost: 0,
        stock: 0,
        unit_of_measure_id: null,
        tax_class_id: null,
        weight_grams: null,
        description: "",
        status: "active",
      });
      toast.success(labels.consignmentProductCreated);
      oncreated?.({ ...product, price: form.price });
    } catch (e: unknown) {
      toast.error(getApiErrorMessage(e, labels.consignmentProductCreateError));
    } finally {
      saving = false;
    }
  }

  $effect(() => {
    if (open) {
      form.sku = "";
      form.name = "";
      form.price = 0;
      loadSku();
    }
  });
</script>

<Modal bind:open title={labels.consignmentNewProduct} size="sm" zIndex={80}>
  <div class="space-y-4">
    <label
      class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary"
    >
      <span>SKU <span class="text-danger">*</span></span>
      <Input
        bind:value={form.sku}
        placeholder={labels.consignmentSkuPlaceholder}
        disabled={skuLoading}
        class="h-9 text-sm"
      />
    </label>
    <label
      class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary"
    >
      <span
        >{labels.consignmentProductName}
        <span class="text-danger">*</span></span
      >
      <Input
        bind:value={form.name}
        placeholder={labels.consignmentProductNamePlaceholder}
        class="h-9 text-sm"
      />
    </label>
    <label
      class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary"
    >
      <span
        >{labels.consignmentProductPrice} (Rp)
        <span class="text-danger">*</span></span
      >
      <NumberInput min="0" bind:value={form.price} class="h-9 text-sm" />
    </label>
  </div>
  {#snippet footer()}
    <div class="flex justify-end gap-3 w-full">
      <Button variant="secondary" onclick={() => (open = false)}>
        {labels.cancel}
      </Button>
      <Button onclick={submit} disabled={saving}>
        {saving ? labels.saving : labels.create}
      </Button>
    </div>
  {/snippet}
</Modal>
