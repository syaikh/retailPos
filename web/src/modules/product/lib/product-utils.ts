import { formatCurrency as _formatCurrency } from "$shared/utils/currency";
import { formatDateTimeInJakarta } from "$shared/utils/jakartaTime";

export type StatusVariant = "success" | "muted" | "danger" | "warning";

export interface StatusInfo {
  variant: StatusVariant;
  label: string;
}

export function statusInfo(status?: string): StatusInfo {
  switch ((status || "").toLowerCase()) {
    case "active":
      return { variant: "success", label: "Active" };
    case "draft":
    case "inactive":
      return {
        variant: "muted",
        label:
          (status || "Draft").charAt(0).toUpperCase() +
          (status || "draft").slice(1),
      };
    case "discontinued":
    case "archived":
      return {
        variant: "danger",
        label: status!.charAt(0).toUpperCase() + status!.slice(1),
      };
    default:
      return { variant: "muted", label: "- " };
  }
}

export function formatCurrency(value?: number): string {
  return _formatCurrency(value, "-");
}

export function formatDate(value?: string): string {
  if (!value) return "-";
  return formatDateTimeInJakarta(value);
}
