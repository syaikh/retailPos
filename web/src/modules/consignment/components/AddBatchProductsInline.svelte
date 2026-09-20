<script lang="ts">
  import { Button, Input, Modal, NumberInput } from "$shared/ui";
  import { Plus, Trash2 } from "lucide-svelte";
  import { toast } from "$shared/stores/toast.svelte";
  import { getApiErrorMessage } from "$shared/utils/error-utils";
  import { labels } from "$shared/i18n";
  import {
    createProduct,
    getNextSku,
  } from "$modules/product/services/product-service";

  interface ProductRow {
    sku: string;
    name: string;
    price: number;
  }

  let {
    open = $bindable(false),
    oncreated,
  }: {
    open: boolean;
    oncreated?: (
      products: { id: number; sku: string; name: string; price: number }[],
    ) => void;
  } = $props();

  let saving = $state(false);
  let rows = $state<ProductRow[]>([{ sku: "", name: "", price: 0 }]);

  const validRows = $derived(rows.filter((r) => r.name.trim()));
  const canSubmit = $derived(validRows.length > 0 && !saving);

  async function loadSku(): Promise<string> {
    try {
      return await getNextSku();
    } catch {
      return "";
    }
  }

  async function addRow() {
    const newSku = await loadSku();
    rows = [...rows, { sku: newSku, name: "", price: 0 }];
  }

  function removeRow(index: number) {
    if (rows.length > 1) {
      rows = rows.filter((_, i) => i !== index);
    }
  }

  function updateRow(
    index: number,
    field: keyof ProductRow,
    value: string | number,
  ) {
    rows = rows.map((row, i) =>
      i === index ? { ...row, [field]: value } : row,
    );
  }

  async function submit() {
    if (validRows.length === 0) {
      toast.error(labels.consignmentAtLeastOneProduct);
      return;
    }

    for (const row of validRows) {
      if (!row.sku.trim()) {
        toast.error(labels.consignmentSkuRequired);
        return;
      }
      if (row.price < 0) {
        toast.error(labels.consignmentPriceNotNegative);
        return;
      }
    }

    saving = true;
    const created: { id: number; sku: string; name: string; price: number }[] =
      [];

    try {
      for (const row of validRows) {
        const product = await createProduct({
          sku: row.sku.trim(),
          name: row.name.trim(),
          barcode: null,
          category: "",
          brand_id: null,
          price: row.price,
          cost: 0,
          stock: 0,
          unit_of_measure_id: null,
          tax_class_id: null,
          weight_grams: null,
          description: "",
          status: "active",
        });
        created.push({ ...product, price: row.price });
      }
      toast.success(
        labels.consignmentProductsCreated.replace(
          "{count}",
          String(created.length),
        ),
      );
      oncreated?.(created);
    } catch (e: unknown) {
      toast.error(getApiErrorMessage(e, labels.consignmentProductCreateError));
    } finally {
      saving = false;
    }
  }

  $effect(() => {
    if (open) {
      rows.splice(0, rows.length, { sku: "", name: "", price: 0 });
      loadSku().then((sku) => {
        if (sku && rows[0] && rows[0].sku === "") rows[0].sku = sku;
      });
    }
  });
</script>

<Modal
  bind:open
  title={labels.consignmentNewProducts}
  size="xl"
  panelClass="max-h-[92vh]"
  zIndex={80}
>
  <div class="space-y-4">
    <p class="text-sm text-text-secondary">
      {labels.consignmentBatchProductHint}
    </p>

    <div class="border border-border rounded-lg overflow-hidden">
      <table class="w-full text-sm">
        <thead class="bg-muted/50">
          <tr
            class="text-left text-xs uppercase tracking-wider text-text-secondary"
          >
            <th class="p-3 w-48">SKU</th>
            <th class="p-3">{labels.consignmentProductName}</th>
            <th class="p-3 w-36">{labels.consignmentProductPrice} (Rp)</th>
            <th class="p-3 w-12"></th>
          </tr>
        </thead>
        <tbody>
          {#each rows as row, i (i)}
            <tr class="border-t border-border">
              <td class="p-2">
                <Input
                  value={row.sku}
                  oninput={(e: Event) =>
                    updateRow(i, "sku", (e.target as HTMLInputElement).value)}
                  placeholder="SKU"
                  class="h-8 text-sm"
                />
              </td>
              <td class="p-2">
                <Input
                  value={row.name}
                  oninput={(e: Event) =>
                    updateRow(i, "name", (e.target as HTMLInputElement).value)}
                  placeholder={labels.consignmentProductNamePlaceholder}
                  class="h-8 text-sm"
                />
              </td>
              <td class="p-2">
                <NumberInput
                  min="0"
                  value={row.price}
                  oninput={(e: Event) =>
                    updateRow(
                      i,
                      "price",
                      Number((e.target as HTMLInputElement).value),
                    )}
                  class="h-8 text-sm"
                />
              </td>
              <td class="p-2">
                <button
                  type="button"
                  onclick={() => removeRow(i)}
                  disabled={rows.length === 1}
                  class="p-1 text-text-muted hover:text-danger disabled:opacity-30"
                >
                  <Trash2 size={14} />
                </button>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>

    <button
      type="button"
      onclick={addRow}
      class="flex items-center gap-1.5 text-sm text-primary hover:text-primary/80"
    >
      <Plus size={14} />
      {labels.consignmentAddRow}
    </button>

    <p class="text-xs text-text-muted">
      {labels.consignmentBatchProductCostHint}
    </p>
  </div>
  {#snippet footer()}
    <div class="flex justify-end gap-3 w-full">
      <Button variant="secondary" onclick={() => (open = false)}>
        {labels.cancel}
      </Button>
      <Button onclick={submit} disabled={!canSubmit}>
        {saving
          ? labels.saving
          : labels.consignmentProductsCreated.replace(
              "{count}",
              String(validRows.length),
            )}
      </Button>
    </div>
  {/snippet}
</Modal>
