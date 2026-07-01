export type ImportStatus = 'queued' | 'parsing' | 'validating' | 'preview_ready' | 'confirmed' | 'importing' | 'completed' | 'failed' | 'cancelled';

export type RowStatus = 'insert' | 'update' | 'error';

export interface PreviewRow {
  rowNumber: number;
  status: RowStatus;
  oldValues: Record<string, string> | null;
  newValues: Record<string, string>;
  errors: ValidationError[];
}

export interface ValidationError {
  row: number;
  field: string;
  value: string;
  reason: string;
  suggestion: string;
  stage: string;
}

export interface PreviewResult {
  module: string;
  total_rows: number;
  insert_count: number;
  update_count: number;
  error_count: number;
  rows: PreviewRow[];
  token?: string;
}

export interface ImportResult {
  job_id: number;
  module: string;
  status: ImportStatus;
  total_rows: number;
  inserted: number;
  updated: number;
  skipped: number;
  errors: number;
  duration_ms: number;
  error_report: string;
}

export interface ImportProgress {
  job_id: number;
  status: ImportStatus;
  progress_pct: number;
  total_rows: number;
  processed: number;
  inserted?: number;
  updated?: number;
  skipped?: number;
  errors: number;
  started_at: string;
  duration_ms: number;
  error_report?: string;
}

export interface ModuleInfo {
  name: string;
  displayName: string;
  features: {
    importEnabled: boolean;
    exportEnabled: boolean;
    templateEnabled: boolean;
  };
}

export type ExportFormat = 'csv' | 'xlsx';
