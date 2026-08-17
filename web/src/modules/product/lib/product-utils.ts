import { formatCurrency as _formatCurrency } from '$shared/utils/currency';
import { formatDateTimeInJakarta } from '$shared/utils/jakartaTime';

export type StatusVariant = 'success' | 'muted' | 'danger' | 'warning';

export interface StatusInfo {
  variant: StatusVariant;
  label: string;
}

export function statusInfo(status?: string): StatusInfo {
  switch ((status || '').toLowerCase()) {
    case 'active':
      return { variant: 'success', label: 'Active' };
    case 'draft':
    case 'inactive':
      return {
        variant: 'muted',
        label: (status || 'Draft').charAt(0).toUpperCase() + (status || 'draft').slice(1),
      };
    case 'discontinued':
    case 'archived':
      return {
        variant: 'danger',
        label: status!.charAt(0).toUpperCase() + status!.slice(1),
      };
    default:
      return { variant: 'muted', label: '- ' };
  }
}

export function formatCurrency(value?: number): string {
  return _formatCurrency(value, '-');
}

export function formatDate(value?: string): string {
  if (!value) return '-';
  return formatDateTimeInJakarta(value);
}

export function validateProductForm(form: {
  name: string;
  sku: string;
  category: string;
  price: number;
  stock: number;
}): string | null {
  if (!form.name.trim() || !form.sku.trim() || !form.category.trim()) {
    return 'Please complete all required fields';
  }
  if (form.price <= 0) {
    return 'Price must be greater than zero';
  }
  if (form.stock < 0) {
    return 'Stock must not be negative';
  }
  return null;
}

export function buildProductPayload(form: {
  name: string;
  sku: string;
  barcode: string;
  category: string;
  brand_id: number | null;
  price: number;
  cost: number;
  stock: number;
  unit_of_measure_id: number | null;
  tax_class_id: number | null;
  weight_grams: number | null;
  description: string;
  status: string;
}) {
  return {
    ...form,
    category_name: form.category,
    barcode: form.barcode?.trim() || undefined,
    description: form.description?.trim() || undefined,
    cost: form.cost >= 0 ? form.cost : undefined,
    weight_grams: form.weight_grams ?? undefined,
  };
}
