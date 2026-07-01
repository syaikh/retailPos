package importexportshared

type ErrorStage string

const (
	StageTemplate  ErrorStage = "template"
	StageType      ErrorStage = "type"
	StageReference ErrorStage = "reference"
	StageBusiness  ErrorStage = "business_rule"
	StageDatabase  ErrorStage = "database"
	StageUnknown   ErrorStage = "unexpected"
)

type ValidationError struct {
	Row        int        `json:"row"`
	Field      string     `json:"field,omitempty"`
	Value      string     `json:"value,omitempty"`
	Reason     string     `json:"reason"`
	Suggestion string     `json:"suggestion,omitempty"`
	Stage      ErrorStage `json:"stage"`
}
