package importexport

type ReferenceItem struct {
	Key   string
	Value interface{}
}

type PreviewRow struct {
	RowNumber int            `json:"row_number"`
	Status    string         `json:"status"`
	OldValues map[string]interface{} `json:"old_values,omitempty"`
	NewValues map[string]interface{} `json:"new_values"`
	Errors    []ValidationError      `json:"errors,omitempty"`
}

type PreviewResult struct {
	Module      string       `json:"module"`
	TotalRows   int          `json:"total_rows"`
	InsertCount int          `json:"insert_count"`
	UpdateCount int          `json:"update_count"`
	ErrorCount  int          `json:"error_count"`
	Rows        []PreviewRow `json:"rows"`
	Token       string       `json:"token,omitempty"`
}

type ImportResult struct {
	JobID       int64  `json:"job_id"`
	Module      string `json:"module"`
	Status      string `json:"status"`
	TotalRows   int    `json:"total_rows"`
	Inserted    int    `json:"inserted"`
	Updated     int    `json:"updated"`
	Skipped     int    `json:"skipped"`
	Errors      int    `json:"errors"`
	DurationMs  int    `json:"duration_ms,omitempty"`
	ErrorReport string `json:"error_report,omitempty"`
}

type ImportProgress struct {
	JobID       int64  `json:"job_id"`
	Status      string `json:"status"`
	ProgressPct int    `json:"progress_pct"`
	TotalRows   int    `json:"total_rows"`
	Processed   int    `json:"processed"`
	Errors      int    `json:"errors"`
	StartedAt   string `json:"started_at"`
	DurationMs  int    `json:"duration_ms,omitempty"`
}
