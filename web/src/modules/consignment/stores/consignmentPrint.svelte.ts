import { writable } from "svelte/store";
import type { ConsignmentReturn, Receipt } from "../types";

export type ConsignmentPrintDoc =
  | { kind: "receipt"; doc: Receipt }
  | { kind: "return"; doc: ConsignmentReturn };

export const consignmentPrint = writable<ConsignmentPrintDoc | null>(null);
