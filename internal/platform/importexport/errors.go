package importexport

import importexportshared "retail-pos-system/internal/shared/importexport"

type ImportError struct {
	JobID      int64                         `json:"job_id"`
	Row        int                           `json:"row"`
	Field      string                        `json:"field,omitempty"`
	Value      string                        `json:"value,omitempty"`
	Reason     string                        `json:"reason"`
	Suggestion string                        `json:"suggestion,omitempty"`
	Stage      importexportshared.ErrorStage `json:"stage"`
}
