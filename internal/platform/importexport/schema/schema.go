package schema

import importexportshared "retail-pos-system/internal/shared/importexport"

type (
	ColumnType      = importexportshared.ColumnType
	ReferencePolicy = importexportshared.ReferencePolicy
	ColumnSchema    = importexportshared.ColumnSchema
	ReferenceDef    = importexportshared.ReferenceDef
	ModuleFeatures  = importexportshared.ModuleFeatures
	ModuleSchema    = importexportshared.ModuleSchema
)

var (
	ColString    = importexportshared.ColString
	ColNumber    = importexportshared.ColNumber
	ColBoolean   = importexportshared.ColBoolean
	ColDate      = importexportshared.ColDate
	ColReference = importexportshared.ColReference

	RefStrict     = importexportshared.RefStrict
	RefAutoCreate = importexportshared.RefAutoCreate
	RefIgnore     = importexportshared.RefIgnore

	IntPtr     = importexportshared.IntPtr
	Float64Ptr = importexportshared.Float64Ptr
)
