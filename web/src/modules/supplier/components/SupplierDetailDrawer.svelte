<script lang="ts">
  import { Drawer, Button, Badge, Skeleton } from '$shared/ui';
  import { Pencil, Trash2 } from 'lucide-svelte';
  import type { Supplier, ProductSupplier } from '../types';
  import { getProductsBySupplier } from '../services/supplier-service';

  function timeAgo(dateStr: string | undefined): string {
    if (!dateStr) return '-';
    const date = new Date(dateStr);
    const now = new Date();
    const seconds = Math.floor((now.getTime() - date.getTime()) / 1000);
    if (seconds < 60) return 'just now';
    const minutes = Math.floor(seconds / 60);
    if (minutes < 60) return `${minutes}m ago`;
    const hours = Math.floor(minutes / 60);
    if (hours < 24) return `${hours}h ago`;
    const days = Math.floor(hours / 24);
    if (days < 30) return `${days}d ago`;
    const months = Math.floor(days / 30);
    if (months < 12) return `${months}mo ago`;
    const years = Math.floor(months / 12);
    return `${years}y ago`;
  }

  let {
    open = $bindable(false),
    supplier = null,
    canEdit = false,
    canDelete = false,
    onclose = () => {},
    onedit = () => {},
    ondelete = () => {},
    onviewproducts = () => {},
  }: {
    open?: boolean;
    supplier?: Supplier | null;
    canEdit?: boolean;
    canDelete?: boolean;
    onclose?: () => void;
    onedit?: (supplier: Supplier) => void;
    ondelete?: (supplier: Supplier) => void;
    onviewproducts?: (supplier: Supplier) => void;
  } = $props();

  let products = $state<ProductSupplier[]>([]);
  let loadingProducts = $state(false);

  $effect(() => {
    if (open && supplier) {
      loadProducts();
    }
  });

  async function loadProducts() {
    if (!supplier) return;
    loadingProducts = true;
    try {
      products = await getProductsBySupplier(supplier.id);
    } catch {
      products = [];
    } finally {
      loadingProducts = false;
    }
  }

  function getInitials(name: string): string {
    return name
      .split(' ')
      .map(w => w[0])
      .join('')
      .toUpperCase()
      .slice(0, 2);
  }

  function handleClose() {
    onclose();
  }
</script>

<Drawer bind:open title={supplier?.name || 'Supplier Details'} onclose={handleClose}>
  {#if supplier}
    <div class="space-y-6">
      <div class="flex items-start gap-4">
        <div class="w-16 h-16 rounded-xl bg-primary-subtle flex items-center justify-center text-primary-light text-xl font-bold shrink-0">
          {getInitials(supplier.name)}
        </div>
        <div class="flex-1 min-w-0">
          <h3 class="text-lg font-semibold text-text-primary">{supplier.name}</h3>
          <p class="text-sm text-text-muted">Code: {supplier.code}</p>
          <div class="mt-2">
            <Badge variant={supplier.is_active ? 'success' : 'muted'}>
              {supplier.is_active ? 'Active' : 'Inactive'}
            </Badge>
          </div>
        </div>
      </div>

      <div class="space-y-4">
        <div>
          <h4 class="text-sm font-medium text-text-muted mb-2">Contact Information</h4>
          <div class="space-y-2 text-sm">
            {#if supplier.contact_name}
              <div class="flex justify-between">
                <span class="text-text-muted">Contact Person</span>
                <span class="text-text-secondary">{supplier.contact_name}</span>
              </div>
            {/if}
            {#if supplier.phone}
              <div class="flex justify-between">
                <span class="text-text-muted">Phone</span>
                <span class="text-text-secondary">{supplier.phone}</span>
              </div>
            {/if}
            {#if supplier.email}
              <div class="flex justify-between">
                <span class="text-text-muted">Email</span>
                <span class="text-text-secondary">{supplier.email}</span>
              </div>
            {/if}
            {#if supplier.address}
              <div class="flex justify-between">
                <span class="text-text-muted">Address</span>
                <span class="text-text-secondary text-right max-w-[200px]">{supplier.address}</span>
              </div>
            {/if}
          </div>
        </div>

        {#if supplier.notes}
          <div>
            <h4 class="text-sm font-medium text-text-muted mb-2">Notes</h4>
            <p class="text-sm text-text-secondary">{supplier.notes}</p>
          </div>
        {/if}

        <div>
          <h4 class="text-sm font-medium text-text-muted mb-2">Linked Products</h4>
          {#if loadingProducts}
            <div class="space-y-2">
              {#each Array(3) as _}
                <Skeleton class="h-8 w-full" />
              {/each}
            </div>
          {:else if products.length === 0}
            <p class="text-sm text-text-muted">No products linked</p>
          {:else}
            <div class="space-y-2">
              {#each products.slice(0, 5) as ps}
                <div class="flex items-center justify-between p-2 bg-surface-subtle rounded-lg">
                  <div class="min-w-0">
                    <p class="text-sm font-medium text-text-primary truncate">{ps.product_name || `Product #${ps.product_id}`}</p>
                    {#if ps.product_sku}
                      <p class="text-xs text-text-muted">SKU: {ps.product_sku}</p>
                    {/if}
                  </div>
                  {#if ps.is_preferred}
                    <Badge variant="primary" class="shrink-0">Preferred</Badge>
                  {/if}
                </div>
              {/each}
              {#if products.length > 0}
                <Button variant="ghost" size="sm" class="w-full" onclick={() => onviewproducts(supplier)}>
                  View all {products.length} products →
                </Button>
              {/if}
            </div>
          {/if}
        </div>

        <div>
          <h4 class="text-sm font-medium text-text-muted mb-2">Timestamps</h4>
          <div class="space-y-2 text-sm">
            <div class="flex justify-between">
              <span class="text-text-muted">Created</span>
              <span class="text-text-secondary">{timeAgo(supplier.created_at)}</span>
            </div>
            <div class="flex justify-between">
              <span class="text-text-muted">Updated</span>
              <span class="text-text-secondary">{timeAgo(supplier.updated_at)}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  {/if}

  {#snippet footer()}
    {#if canEdit}
      <Button variant="secondary" onclick={() => onedit(supplier!)}>
        <Pencil class="w-4 h-4 mr-2" />
        Edit
      </Button>
    {/if}
    {#if canDelete}
      <Button variant="danger" onclick={() => ondelete(supplier!)}>
        <Trash2 class="w-4 h-4 mr-2" />
        Delete
      </Button>
    {/if}
  {/snippet}
</Drawer>