import { type DateValue, CalendarDate } from "@internationalized/date";

export const cn = (...args: any[]) => args.filter(Boolean).join(" ");

export type Theme = {
  bg?: string;
  text?: string;
  muted?: string;
  border?: string;
  hover?: string;
  selected?: string;
  selectedText?: string;
  todayBorder?: string;
  radius?: string;
};

export const defaultTheme: Theme = {
  bg: "#ffffff",
  text: "#1a1a2e",
  muted: "#64748b",
  border: "#e2e8f0",
  hover: "#e0f2fe",
  selected: "#0284c7",
  selectedText: "#ffffff",
  todayBorder: "#0ea5e9",
  radius: "8px",
};

export const getThemeStyle = (theme: Theme = {}) => {
  const text = theme.text || defaultTheme.text;
  const muted = theme.muted || defaultTheme.muted;
  const border = theme.border || defaultTheme.border;
  const hover = theme.hover || defaultTheme.hover;
  const selected = theme.selected || defaultTheme.selected;
  const selectedText = theme.selectedText || defaultTheme.selectedText;
  const todayBorder = theme.todayBorder || defaultTheme.todayBorder;
  const bg = theme.bg || defaultTheme.bg;
  const radius = theme.radius || defaultTheme.radius;
  
  return `
    --calendar-text: ${text};
    --calendar-muted: ${muted};
    --calendar-border: ${border};
    --calendar-hover: ${hover};
    --calendar-selected: ${selected};
    --calendar-selected-text: ${selectedText};
    --calendar-today-border: ${todayBorder};
    --calendar-bg: ${bg};
    --calendar-radius: ${radius};
    background: ${bg};
    border-radius: ${radius};
  `;
};