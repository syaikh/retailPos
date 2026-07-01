package validation

import (
	"context"

	importexportshared "retail-pos-system/internal/shared/importexport"
	"retail-pos-system/internal/platform/importexport/validation/validators"
)

type Pipeline struct {
	validators []Validator
}

func NewDefaultPipeline() *Pipeline {
	return &Pipeline{
		validators: []Validator{
			&validators.FileValidator{},
			&validators.TemplateValidator{},
			&validators.TypeValidator{},
			&validators.RequiredValidator{},
			&validators.ReferenceValidator{},
			&validators.DuplicateValidator{},
		},
	}
}

func (p *Pipeline) Add(v Validator) {
	p.validators = append(p.validators, v)
}

func (p *Pipeline) Run(ctx context.Context, s importexportshared.ModuleSchema, rows []map[string]interface{}, refs map[string][]importexportshared.ReferenceItem) []importexportshared.ValidationError {
	var allErrors []importexportshared.ValidationError
	for _, v := range p.validators {
		errs := v.Validate(ctx, s, rows, refs)
		allErrors = append(allErrors, errs...)
	}
	return allErrors
}
