import { getTodayInJakarta } from "$shared/utils/jakartaTime";
import { formatLocaleDate } from "$shared/i18n";

export function formatCurrencyShort(value: number | null | undefined): string {
  if (value == null) return "Rp 0";
  if (value >= 1000000000)
    return "Rp " + (value / 1000000000).toFixed(1).replace(/\.0$/, "") + "M";
  if (value >= 1000000)
    return "Rp " + (value / 1000000).toFixed(1).replace(/\.0$/, "") + "jt";
  if (value >= 1000) return "Rp " + (value / 1000).toFixed(0) + "k";
  return "Rp " + value.toLocaleString("id-ID");
}

export function formatLargeNumber(value: number | null | undefined): string {
  if (value == null) return "0";
  if (value >= 1000000000)
    return (value / 1000000000).toFixed(1).replace(/\.0$/, "") + "M";
  if (value >= 1000000)
    return (value / 1000000).toFixed(1).replace(/\.0$/, "") + "jt";
  if (value >= 1000) return (value / 1000).toFixed(0) + "k";
  return value.toLocaleString("id-ID");
}

export function formatDate(dateString?: string): string {
  if (!dateString) return "";
  const date = new Date(dateString + "T00:00:00Z");
  const day = date.getUTCDate().toString().padStart(2, "0");
  const month = formatLocaleDate(date, { month: "short", timeZone: "UTC" });
  const year = date.getUTCFullYear();
  return `${day} ${month} ${year}`;
}

export function getPeriodLabel(item: {
  hour?: number;
  date?: string;
  month_start?: string;
  label?: string;
}): string {
  if (!item) return "";
  if (item.hour !== undefined)
    return `${String(item.hour).padStart(2, "0")}:00`;
  if (item.date) {
    if (/^\d{1,2}$/.test(item.date)) {
      return `${item.date.padStart(2, "0")}:00`;
    }
    const d = new Date(item.date + "T00:00:00Z");
    const day = d.getUTCDate();
    const month = formatLocaleDate(d, { month: "short", timeZone: "UTC" });
    return `${day} ${month}`;
  }
  if (item.month_start) {
    const d = new Date(item.month_start + "T00:00:00Z");
    const month = formatLocaleDate(d, { month: "short", timeZone: "UTC" });
    const year = d.getUTCFullYear();
    return `${month} ${year}`;
  }
  return item.label || "";
}

export function formatDayDate(dateString?: string): string {
  if (!dateString) return "";
  const date = new Date(dateString + "T00:00:00Z");
  const dayName = formatLocaleDate(date, { weekday: "short", timeZone: "UTC" });
  const day = date.getUTCDate();
  const month = formatLocaleDate(date, { month: "short", timeZone: "UTC" });
  return `${dayName}, ${day} ${month}`;
}

export function getFirstOfMonthNAgoInJakarta(n: number): string {
  const today = getTodayInJakarta().split("-").map(Number);
  const totalMonths = today[0] * 12 + today[1] - 1 - n;
  const year = Math.floor(totalMonths / 12);
  const month = (totalMonths % 12) + 1;
  return `${year}-${String(month).padStart(2, "0")}-01`;
}
