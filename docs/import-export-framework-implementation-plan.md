# Reusable Import & Export Framework — Implementation Plan (Revised)

| Status       | Draft — pending approval         |
|-------------|----------------------------------|
| Priority     | High                             |
| Owner        | Platform Team                    |
| Affected     | Products, Categories, Brands, Units of Measure, Customers |
| Tech Stack   | Go (backend), Svelte 5 + TypeScript + TailwindCSS 4 (frontend) |
| Architecture | **Schema-driven** — ModuleSchema is the central pillar |

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Architecture Principle: Schema-Driven](#2-architecture-principle-schema-driven)
3. [Current State Analysis](#3-current-state-analysis)
4. [High-Level Architecture](#4-high-level-architecture)
5. [ModuleSchema — The Central Pillar](#5-moduleschema--the-central-pillar)
6. [Implementation Steps](#6-implementation-steps)
   - [Step 0: Extract Brand & UOM Modules](#step-0-extract-brand--uom-modules)
   - [Step 1: Database Migration](#step-1-database-migration)
   - [Step 2: ModuleSchema + Schema Registry](#step-2-moduleschema--schema-registry)
   - [Step 3: Pluggable Validator Pipeline](#step-3-pluggable-validator-pipeline)
   - [Step 4: Template Engine](#step-4-template-engine)
   - [Step 5: Import Engine](#step-5-import-engine)
   - [Step 6: Export Engine](#step-6-export-engine)
   - [Step 7: Progress Engine + Cancellation](#step-7-progress-engine--cancellation)
   - [Step 8: Thin Adapters](#step-8-thin-adapters)
   - [Step 9: Frontend Components](#step-9-frontend-components)
   - [Step 10: Module Migration + Cleanup](#step-10-module-migration--cleanup)
   - [Step 11: Performance Optimization](#step-11-performance-optimization)
7. [Interface Definitions](#7-interface-definitions)
8. [History Model](#8-history-model)
9. [Testing Strategy](#9-testing-strategy)
10. [Definition of Done](#10-definition-of-done)

---

## 1. Executive Summary

Replace every existing Import/Export implementation with a reusable **Schema-driven** Import & Export Framework. The framework is a platform-layer service, not owned by any module. It uses `ModuleSchema` as the single source of truth — consumed by validation, template generation, import, and export engines. Adapters are thin bridges to domain logic only.

### Key Architectural Shift

```
BEFORE (Adapter-driven):
  Adapter owns columns, validation, mapping, persistence
  Engine delegates everything to Adapter
  → Fat adapters (800+ lines)
  → Hard to add new modules
  → Schema knowledge scattered

AFTER (Schema-driven):
  ModuleSchema owns columns, types, references, rules
  Engine uses Schema for all generic operations
  Adapter handles ONLY domain-specific logic
  → Thin adapters (~50-100 lines)
  → New module = Schema + thin adapter
  → Single source of truth
```

---

## 2. Architecture Principle: Schema-Driven

### Control Flow

```
                    ModuleSchema
                 (versioned, self-describing,
                  single source of truth)
                        │
                        ▼
                  Schema Registry
                 (module → schema)
              ┌───────┼───────────┐
              │       │           │
              ▼       ▼           ▼
       Validation   Template    Export/Import
       Engine       Engine      Engine
              │       │           │
              └───────┼───────────┘
                      ▼
               Adapters (thin)
                  ~4 methods
                      ▼
                 Repositories
```

### What Schema Provides vs What Adapter Provides

| Concern | Owner | Why |
|---------|-------|-----|
| Column names & types | **Schema** | Structural, reusable across modules |
| Required fields | **Schema** | Declarative, no code needed |
| Reference definitions | **Schema** | Includes policy (strict/auto-create/ignore) |
| Allowed values | **Schema** | Declarative validation |
| Default values | **Schema** | Applied by engine |
| Partial update rules | **Schema** | `Editable` field flag |
| Export formatting | **Schema** | `Exportable` field flag |
| Template layout | **Schema** | Columns → Reference sheets → Dropdowns |
| Type validation | **Engine** (using Schema) | Generic, works for any module |
| Reference validation | **Engine** (using Schema + policy) | Policy-driven |
| Duplicate detection | **Engine** (using Schema `BusinessKeys`) | Generic |
| Business logic | **Adapter** | Domain-specific (e.g., Price > 0) |
| Entity mapping | **Adapter** | Schema row → domain struct |
| Persistence | **Adapter** | Insert/Update SQL |

---

## 3. Current State Analysis

### Duplication Inventory

| Pattern | Files | Duplicate LOC |
|---------|-------|---------------|
| Export XLSX handler (header style, column widths, content-type) | `product/handler.go` (×3), `category/handler.go`, `customer/handler.go` | ~200 |
| Export CSV handler (BOM, sanitization, writing rows) | Same 5 handlers | ~150 |
| Import handler (file read, column index lookup, row parsing) | Same 5 handlers | ~250 |
| `BulkUpsert*` repository methods (per-row SQL upsert loop) | `product/repo.go` (×3), `category/repo.go`, `customer/repo.go` | ~200 |
| `GetAll*ForExport` repository queries | Same 5 locations | ~100 |
| Service passthrough methods | Same 5 services | ~50 |
| **Total** | | **~950 lines** |

### Missing Features

| Feature | Current | Target |
|---------|---------|--------|
| Preview | None | Mandatory, per-row status + diff |
| Validation | Inline fail-fast | Multi-stage pipeline, error accumulation |
| Import format | CSV only | CSV + XLSX |
| Template | Client-side CSV headers | Server XLSX + reference sheets + dropdowns |
| Reference validation | Auto-creates missing | Policy-driven (strict/auto-create/ignore) |
| Import history | None | Job → Snapshot → Rows → Errors |
| Error reports | Inline strings | Downloadable XLSX |
| Transaction | Per-row | Single transaction per import |
| SQL batching | One query per row | Batch operations |
| Progress tracking | None | Status lifecycle + polling |
| Cancellation | None | Context-based cancel |
| Schema versioning | None | Semver, stored in history |

---

## 4. High-Level Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│                        Svelte Frontend                           │
│  ImportWizard  PreviewTable  HistoryDialog  ValidationSummary    │
│  DropZone  ProgressDialog  ImportSummary  BulkActionDropdown      │
└──────────────────────────┬───────────────────────────────────────┘
                           │ HTTP
┌──────────────────────────▼───────────────────────────────────────┐
│                    Generic API Handlers                          │
│  POST /{module}/import/preview  POST /{module}/import            │
│  GET  /{module}/export          GET  /{module}/template          │
│  GET  /{module}/import/history  GET  /{module}/import/history/:id│
│  POST /{module}/import/{id}/cancel                               │
│  GET  /{module}/import/{id}/progress                             │
└──────────────────────────┬───────────────────────────────────────┘
                           │
┌──────────────────────────▼───────────────────────────────────────┐
│                    Schema Registry                               │
│         module → ModuleSchema + Adapter                          │
└──────┬──────────┬──────────┬──────────┬──────────┬───────────────┘
       │          │          │          │          │
       ▼          ▼          ▼          ▼          ▼
   Validation  Template   Import     Export    Progress
   Engine      Engine     Engine     Engine    Engine
       │          │          │          │          │
       └──────────┴──────────┼──────────┴──────────┘
                             ▼
                     Adapters (thin)
                     ┌─────┼─────┐
                     │     │     │
                     ▼     ▼     ▼
                 Product Brand Category UOM Customer
                 Repo   Repo  Repo      Repo Repo
```

---

## 5. ModuleSchema — The Central Pillar

### Definition

```go
type ModuleSchema struct {
    ModuleName    string           `json:"module_name"`
    DisplayName   string           `json:"display_name"`
    SchemaVersion string           `json:"schema_version"`    // semver, e.g. "1.0.0"
    Description   string           `json:"description"`
    Columns       []ColumnSchema   `json:"columns"`
    PrimaryKey    string           `json:"primary_key"`
    BusinessKeys  []string         `json:"business_keys"`     // uniqueness constraints
    References    []ReferenceDef   `json:"references"`
    Features      ModuleFeatures   `json:"features"`
}

type ColumnSchema struct {
    Name          string       `json:"name"`
    Type          ColumnType   `json:"type"`       // string, number, boolean, date, reference
    Label         string       `json:"label"`      // human-readable display name
    Required      bool         `json:"required"`
    MaxLength     *int         `json:"max_length,omitempty"`
    MinValue      *float64     `json:"min_value,omitempty"`
    MaxValue      *float64     `json:"max_value,omitempty"`
    AllowedValues []string     `json:"allowed_values,omitempty"`
    Reference     string       `json:"reference,omitempty"`     // reference module name
    Default       interface{}  `json:"default,omitempty"`
    Description   string       `json:"description,omitempty"`
    Editable      bool         `json:"editable"`      // import can update this field
    Exportable    bool         `json:"exportable"`    // included in export
    Template      bool         `json:"template"`      // included in template
    ImportGroup   string       `json:"import_group,omitempty"`  // logical grouping for validation
}

type ColumnType string
const (
    ColString   ColumnType = "string"
    ColNumber   ColumnType = "number"
    ColBoolean  ColumnType = "boolean"
    ColDate     ColumnType = "date"
    ColReference ColumnType = "reference"
)

type ReferenceDef struct {
    Column          string          `json:"column"`            // column name in this schema
    ReferenceModule string          `json:"reference_module"`  // e.g. "brands"
    ReferenceColumn string          `json:"reference_column"`  // e.g. "Name"
    ReferenceLabel  string          `json:"reference_label"`   // e.g. "Brand Name"
    Policy          ReferencePolicy `json:"policy"`
}

type ReferencePolicy string
const (
    RefStrict     ReferencePolicy = "strict"       // fail if reference missing
    RefAutoCreate ReferencePolicy = "auto_create"  // create if missing
    RefIgnore     ReferencePolicy = "ignore"       // skip reference validation
)

type ModuleFeatures struct {
    ImportEnabled   bool `json:"import_enabled"`
    ExportEnabled   bool `json:"export_enabled"`
    TemplateEnabled bool `json:"template_enabled"`
    PartialUpdate   bool `json:"partial_update"`
    MassInsert      bool `json:"mass_insert"`
    MassUpdate      bool `json:"mass_update"`
    SupportsPreview bool `json:"supports_preview"`     // default true
    SupportsHistory bool `json:"supports_history"`     // default true
}
```

### Example: ProductSchema

```go
var ProductSchema = ModuleSchema{
    ModuleName:    "products",
    DisplayName:   "Products",
    SchemaVersion: "1.0.0",
    Description:   "Master product data",
    PrimaryKey:    "id",
    BusinessKeys:  []string{"SKU"},
    Columns: []ColumnSchema{
        {Name: "SKU",           Type: ColString,   Label: "SKU",             Required: true,  MaxLength: intPtr(50),  Editable: false, Exportable: true, Template: true},
        {Name: "Name",          Type: ColString,   Label: "Product Name",    Required: true,  MaxLength: intPtr(200), Editable: true,  Exportable: true, Template: true},
        {Name: "Barcode",       Type: ColString,   Label: "Barcode",         Required: false, MaxLength: intPtr(50),  Editable: true,  Exportable: true, Template: true},
        {Name: "Category",      Type: ColReference,Label: "Category",        Required: false, Reference: "categories",Editable: true,  Exportable: true, Template: true},
        {Name: "Brand",         Type: ColReference,Label: "Brand",           Required: false, Reference: "brands",   Editable: true,  Exportable: true, Template: true},
        {Name: "Price",         Type: ColNumber,   Label: "Price",           Required: true,  MinValue: floatPtr(1),  Editable: true,  Exportable: true, Template: true},
        {Name: "Cost",          Type: ColNumber,   Label: "Cost",            Required: false, MinValue: floatPtr(0),  Editable: true,  Exportable: true, Template: true},
        {Name: "Stock",         Type: ColNumber,   Label: "Stock",           Required: false, MinValue: floatPtr(0),  Editable: true,  Exportable: true, Template: false},
        {Name: "Status",        Type: ColString,   Label: "Status",          Required: false, AllowedValues: []string{"active","inactive","draft","archived"}, Default: "active", Editable: true, Exportable: true, Template: true},
        {Name: "UnitOfMeasure", Type: ColReference,Label: "Unit of Measure", Required: false, Reference: "uoms",      Editable: true,  Exportable: true, Template: true},
        {Name: "WeightGrams",   Type: ColNumber,   Label: "Weight (g)",      Required: false, MinValue: floatPtr(0),  Editable: true,  Exportable: true, Template: true},
        {Name: "Description",   Type: ColString,   Label: "Description",     Required: false, MaxLength: intPtr(2000),Editable: true,  Exportable: true, Template: true},
    },
    References: []ReferenceDef{
        {Column: "Category",      ReferenceModule: "categories", ReferenceColumn: "Name", Policy: RefStrict},
        {Column: "Brand",         ReferenceModule: "brands",     ReferenceColumn: "Name", Policy: RefStrict},
        {Column: "UnitOfMeasure", ReferenceModule: "uoms",       ReferenceColumn: "Code", Policy: RefStrict},
    },
    Features: ModuleFeatures{
        ImportEnabled:   true,
        ExportEnabled:   true,
        TemplateEnabled: true,
        PartialUpdate:   true,
        MassInsert:      true,
        MassUpdate:      true,
        SupportsPreview: true,
        SupportsHistory: true,
    },
}
```

### Example: CategorySchema

```go
var CategorySchema = ModuleSchema{
    ModuleName:    "categories",
    DisplayName:   "Categories",
    SchemaVersion: "1.0.0",
    PrimaryKey:    "id",
    BusinessKeys:  []string{"Name"},
    Columns: []ColumnSchema{
        {Name: "Name",        Type: ColString,  Label: "Name",        Required: true,  MaxLength: intPtr(100), Editable: true, Exportable: true, Template: true},
        {Name: "Slug",        Type: ColString,  Label: "Slug",        Required: false, MaxLength: intPtr(120), Editable: true, Exportable: true, Template: true},
        {Name: "Description", Type: ColString,  Label: "Description", Required: false, MaxLength: intPtr(500), Editable: true, Exportable: true, Template: true},
        {Name: "IsActive",    Type: ColBoolean, Label: "Active",      Required: false, Default: true,          Editable: true, Exportable: true, Template: true},
    },
    Features: ModuleFeatures{
        ImportEnabled:   true,
        ExportEnabled:   true,
        TemplateEnabled: true,
        PartialUpdate:   true,
        MassInsert:      true,
        MassUpdate:      true,
        SupportsPreview: true,
        SupportsHistory: true,
    },
}
```

---

## 6. Implementation Steps

### Step 0: Extract Brand & UOM Modules

**Commit:** `refactor: extract Brand and UnitOfMeasure into independent modules`

Create `internal/brand/` and `internal/uom/` packages by extracting from `internal/product/`.

| Action | Details |
|--------|---------|
| Create `internal/brand/` | Domain structs, Repository, Service, Handler |
| Create `internal/uom/` | Domain structs, Repository, Service, Handler |
| Move structs | `Brand`, `BrandImportRow` → `internal/brand/` |
| Move structs | `UnitOfMeasure`, `UnitOfMeasureImportRow` → `internal/uom/` |
| Copy repository + service + handler | CRUD, routes, export, import |
| Update `main.go` | Wire new repos, services, handlers, register routes |
| Update `internal/product/` | Remove Brand/UOM code, accept brand/uom services as deps |
| Remove `GetOrCreate*` from product repo | No more auto-creation (will be enforced in Step 3) |

**No DB migration needed** — data stays in same tables.

---

### Step 1: Database Migration

**Commit:** `feat: add import history tables (jobs, snapshots, rows, errors)`

```sql
CREATE TABLE import_jobs (
    id              BIGSERIAL    PRIMARY KEY,
    module          VARCHAR(50)  NOT NULL,
    schema_version  VARCHAR(20)  NOT NULL,
    filename        VARCHAR(255) NOT NULL,
    status          VARCHAR(20)  NOT NULL DEFAULT 'queued',
    -- queued → parsing → validating → preview_ready → confirmed → importing → completed
    --                                                                    → failed
    --                                                                    → cancelled
    total_rows      INT          NOT NULL DEFAULT 0,
    inserted        INT          NOT NULL DEFAULT 0,
    updated         INT          NOT NULL DEFAULT 0,
    skipped         INT          NOT NULL DEFAULT 0,
    error_count     INT          NOT NULL DEFAULT 0,
    progress_pct    INT          NOT NULL DEFAULT 0,
    error_report_path VARCHAR(500),
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    duration_ms     INT,
    user_id         INT          NOT NULL REFERENCES users(id),
    store_id        INT          REFERENCES stores(id),
    cancel_requested BOOLEAN     NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE import_snapshots (
    id              BIGSERIAL    PRIMARY KEY,
    import_job_id   BIGINT       NOT NULL REFERENCES import_jobs(id) ON DELETE CASCADE,
    rows_data       JSONB        NOT NULL,  -- full snapshot of all parsed rows at preview time
    schema_snapshot JSONB        NOT NULL,  -- copy of ModuleSchema at time of import
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE import_rows (
    id              BIGSERIAL    PRIMARY KEY,
    import_job_id   BIGINT       NOT NULL REFERENCES import_jobs(id) ON DELETE CASCADE,
    row_number      INT          NOT NULL,
    status          VARCHAR(20)  NOT NULL,  -- inserted, updated, skipped, error
    entity_id       INT,
    old_values      JSONB,
    new_values      JSONB,
    changed_fields  TEXT[],
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE import_errors (
    id              BIGSERIAL    PRIMARY KEY,
    import_job_id   BIGINT       NOT NULL REFERENCES import_jobs(id) ON DELETE CASCADE,
    row_number      INT          NOT NULL,
    field           VARCHAR(100),
    value           TEXT,
    reason          TEXT         NOT NULL,
    suggestion      TEXT,
    stage           VARCHAR(30)  NOT NULL,  -- type, reference, business, duplicate
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_import_jobs_module ON import_jobs(module);
CREATE INDEX idx_import_jobs_user   ON import_jobs(user_id);
CREATE INDEX idx_import_jobs_status ON import_jobs(status);
CREATE INDEX idx_import_rows_job    ON import_rows(import_job_id);
CREATE INDEX idx_import_errors_job  ON import_errors(import_job_id);
```

---

### Step 2: ModuleSchema + Schema Registry

**Commit:** `feat: add ModuleSchema and SchemaRegistry`

**New files:**

```
internal/platform/importexport/
├── schema/
│   ├── schema.go          # ModuleSchema, ColumnSchema, ReferenceDef, etc.
│   └── registry.go        # SchemaRegistry — map[module]ModuleSchema
├── interfaces.go          # Adapter interface (thin)
├── errors.go              # ValidationError, ImportError types
└── models.go              # PreviewRow, ImportResult, progress types
```

#### Schema Registry

```go
type SchemaRegistry struct {
    schemas map[string]ModuleSchema
}

func NewSchemaRegistry() *SchemaRegistry
func (r *SchemaRegistry) Register(schema ModuleSchema) error
func (r *SchemaRegistry) Get(module string) (ModuleSchema, error)
func (r *SchemaRegistry) All() []ModuleSchema
```

#### Schema Registration

```go
// In main.go or module init
schemaReg := importexport.NewSchemaRegistry()
schemaReg.Register(category.CategorySchema)
schemaReg.Register(brand.BrandSchema)
schemaReg.Register(uom.UOMSchema)
schemaReg.Register(customer.CustomerSchema)
schemaReg.Register(product.ProductSchema)
```

---

### Step 3: Pluggable Validator Pipeline

**Commit:** `feat: add pluggable validation pipeline`

**New files:**

```
internal/platform/importexport/validation/
├── pipeline.go          # ValidationPipeline — ordered validator execution
├── validator.go         # Validator interface
├── validators/
│   ├── file.go          # FileValidator — format, size, MIME
│   ├── template.go      # TemplateValidator — required columns, version
│   ├── type.go          # TypeValidator — data type per ColumnSchema
│   ├── required.go      # RequiredValidator — Required fields
│   ├── reference.go     # ReferenceValidator — lookup against loaded refs + policy
│   └── duplicate.go     # DuplicateValidator — BusinessKeys uniqueness
```

#### Validator Interface

```go
type Validator interface {
    Name() string
    Validate(ctx context.Context, schema ModuleSchema, rows []map[string]interface{}, refs map[string][]ReferenceItem) []ValidationError
}

type ValidationPipeline struct {
    validators []Validator
}

func NewDefaultPipeline() *ValidationPipeline {
    return &ValidationPipeline{
        validators: []Validator{
            &FileValidator{},
            &TemplateValidator{},
            &TypeValidator{},
            &RequiredValidator{},
            &ReferenceValidator{},
            &DuplicateValidator{},
        },
    }
}

func (p *ValidationPipeline) Add(v Validator) {
    p.validators = append(p.validators, v)
}

func (p *ValidationPipeline) Run(ctx context.Context, schema ModuleSchema, rows []map[string]interface{}, refs map[string][]ReferenceItem) []ValidationError {
    var allErrors []ValidationError
    for _, v := range p.validators {
        errs := v.Validate(ctx, schema, rows, refs)
        allErrors = append(allErrors, errs...)
        // Do NOT fail-fast — collect all errors
    }
    return allErrors
}
```

#### Module-specific Validator Injection

```go
// Product registers a custom business validator
pipeline := validation.NewDefaultPipeline()
pipeline.Add(&ProductBusinessValidator{minPrice: 1})
pipeline.Add(&ProductDuplicateBarcodeValidator{repo: productRepo})

// Category — no custom validators needed
pipeline := validation.NewDefaultPipeline()
```

---

### Step 4: Template Engine

**Commit:** `feat: add schema-driven template engine`

**New files:**

```
internal/platform/importexport/template/
└── engine.go            # TemplateEngine — ModuleSchema → XLSX
```

#### Flow

```
ModuleSchema
    │
    ▼
TemplateEngine
    │
    ├── Create Instruction Sheet (schema description, version, instructions)
    ├── Create Main Data Sheet (columns where Template=true)
    │     ├── Header row (ColumnSchema.Label)
    │     ├── Data validation dropdowns for reference columns
    │     └── Column widths, formatting
    ├── Create Reference Sheet (one per ReferenceDef)
    │     ├── Column: ReferenceColumn (e.g. BrandCode)
    │     └── Column: ReferenceLabel (e.g. BrandName)
    └── Named ranges for dropdown data sources
    │
    ▼
    XLSX file → download
```

No adapter involvement. Schema is the sole input.

---

### Step 5: Import Engine

**Commit:** `feat: add schema-driven import engine`

**New files:**

```
internal/platform/importexport/import/
├── engine.go            # ImportEngine — orchestrates the full import lifecycle
├── parser.go            # FileParser — CSV or XLSX → []map[string]interface{}
└── preview.go           # PreviewGenerator — row diff computation
```

#### Import Lifecycle

```
1. Parse file (CSV or XLSX) → headers + rows
2. Validate template (headers match schema columns)
3. Load references (via adapter)
4. Run validation pipeline (collect ALL errors)
5. Generate preview (insert/update/skip/error per row)
6. Store snapshot (import_snapshots table)
7. Wait for user confirmation
8. Begin transaction
9. Execute: for each row, adapter.Insert() or adapter.Update()
10. Commit or Rollback
11. Record history (import_rows + import_errors)
12. Return summary
```

#### Key: Engine reads Schema, not Adapter

```go
type ImportEngine struct {
    schemaReg  *SchemaRegistry
    adapterReg *AdapterRegistry
    validator  *ValidationPipeline
    progress   *ProgressEngine
    history    *HistoryRepository
}

func (e *ImportEngine) Preview(ctx context.Context, module string, file io.Reader) (*PreviewResult, error) {
    schema := e.schemaReg.Get(module)
    adapter := e.adapterReg.Get(module)
    rows := ParseFile(file, schema)        // uses schema.Columns
    refs := adapter.LoadReferences(ctx)    // only reference data
    errors := e.validator.Run(ctx, schema, rows, refs)
    preview := GeneratePreview(schema, rows, errors)  // uses schema.BusinessKeys
    return preview, nil
}

func (e *ImportEngine) Execute(ctx context.Context, module string, confirmationToken string) (*ImportResult, error) {
    // Load snapshot
    // Get adapter
    // Begin tx
    // For each row: adapter.Insert() or adapter.Update()
    // Commit
    // Record history
}
```

---

### Step 6: Export Engine

**Commit:** `feat: add schema-driven export engine`

**New files:**

```
internal/platform/importexport/export/
└── engine.go            # ExportEngine — ModuleSchema + data → CSV or XLSX
```

#### Flow

```
Adapter.ExportData(schema) → []map[string]interface{}
    │
    ▼
ExportEngine
    │
    ├── Filter columns where Exportable=true
    ├── Apply type formatting (number, boolean, date)
    ├── CSV: write rows with BOM + sanitization
    └── XLSX: styled workbook with headers from schema
```

No duplicate header definitions. Schema is the source.

---

### Step 7: Progress Engine + Cancellation

**Commit:** `feat: add progress tracking and import cancellation`

**New files:**

```
internal/platform/importexport/progress/
└── engine.go            # ProgressEngine — status tracking + polling
```

#### Status Lifecycle

```
queued → parsing → validating → preview_ready
                                    │
                              confirmed → importing → completed
                                             │
                                             ├──→ failed
                                             └──→ cancelled
```

#### Progress Model

```go
type ImportProgress struct {
    JobID       int64  `json:"job_id"`
    Status      string `json:"status"`       // current status
    ProgressPct int    `json:"progress_pct"` // 0-100
    TotalRows   int    `json:"total_rows"`
    Processed   int    `json:"processed"`
    Errors      int    `json:"errors"`
    StartedAt   string `json:"started_at"`
    DurationMs  int    `json:"duration_ms,omitempty"`
}
```

#### Cancellation

```go
// API: POST /api/{module}/import/{id}/cancel
func HandleCancelImport(registry *AdapterRegistry) gin.HandlerFunc {
    return func(c *gin.Context) {
        jobID := c.Param("id")
        // Set cancel_requested = true in import_jobs table
        // Import engine checks ctx.Done() between batches
        // If before transaction: mark as cancelled immediately
        // If during transaction: set flag, rollback on next check
    }
}
```

The import engine checks `ctx.Done()` between batch operations:

```go
for i, batch := range batches {
    select {
    case <-ctx.Done():
        tx.Rollback()
        historyRepo.UpdateStatus(ctx, jobID, "cancelled")
        return nil, ErrCancelled
    default:
        _, err := adapter.Insert(ctx, batch)
        if err != nil {
            tx.Rollback()
            return nil, err
        }
        progress.Update(i, len(batches))
    }
}
```

---

### Step 8: Thin Adapters

**Commit:** `feat: implement thin adapters for all modules`

#### Adapter Interface

```go
type Adapter interface {
    ModuleName() string

    // BusinessValidation performs module-specific business validation.
    // Generic validation (type, reference, duplicate) is handled by the pipeline.
    ValidateBusiness(ctx context.Context, schema ModuleSchema, rows []map[string]interface{}) []ValidationError

    // MapToEntity converts a validated row to the domain entity for persistence.
    MapToEntity(ctx context.Context, schema ModuleSchema, row map[string]interface{}) (interface{}, error)

    // Repository provides data access operations.
    Repository() RepositoryActions
}

type RepositoryActions interface {
    // Insert batch-inserts entities. Returns count.
    Insert(ctx context.Context, entities []interface{}) (int, error)

    // Update batch-updates entities. Returns count.
    Update(ctx context.Context, entities []interface{}) (int, error)

    // ExportData fetches all data as []map[string]interface{}.
    ExportData(ctx context.Context, schema ModuleSchema) ([]map[string]interface{}, error)

    // LoadReferences loads reference data for validation.
    // Called once per import.
    LoadReferences(ctx context.Context, schema ModuleSchema) (map[string][]ReferenceItem, error)
}
```

#### Adapter Registrations

```go
type AdapterRegistry struct {
    adapters map[string]Adapter
}

func NewAdapterRegistry() *AdapterRegistry
func (r *AdapterRegistry) Register(adapter Adapter) error
func (r *AdapterRegistry) Get(module string) (Adapter, error)
```

#### Registration in main.go

```go
schemaReg := importexport.NewSchemaRegistry()
schemaReg.Register(category.Schema)  // ModuleSchema
schemaReg.Register(brand.Schema)
schemaReg.Register(uom.Schema)
schemaReg.Register(customer.Schema)
schemaReg.Register(product.Schema)

adapterReg := importexport.NewAdapterRegistry()
adapterReg.Register(category.NewAdapter(categoryRepo))
adapterReg.Register(brand.NewAdapter(brandRepo))
adapterReg.Register(uom.NewAdapter(uomRepo))
adapterReg.Register(customer.NewAdapter(customerRepo))
adapterReg.Register(product.NewAdapter(productSvc, brandSvc, categorySvc, uomSvc))
```

#### Adapter Size Comparison

| Adapter | Old (estimated) | New (estimated) |
|---------|----------------|-----------------|
| Category | ~80 lines | ~30 lines |
| Brand | ~80 lines | ~30 lines |
| UOM | ~80 lines | ~30 lines |
| Customer | ~80 lines | ~40 lines |
| Product | ~200 lines | ~80-100 lines |

---

### Step 9: Frontend Components

**Commit:** `feat: add reusable import/export frontend components`

**New package tree:**

```
web/src/lib/components/import-export/
├── BulkActionDropdown.svelte    # Export/Template/Import/History dropdown
├── ImportWizard.svelte          # Multi-step wizard (upload → validate → preview → confirm → result)
├── DropZone.svelte              # Drag-and-drop file upload (CSV + XLSX)
├── ValidationSummary.svelte     # Tabular error display with row/field/reason/suggestion
├── PreviewTable.svelte          # Row-by-row diff (insert/update/skip/error, old vs new)
├── ProgressDialog.svelte        # Progress bar + status (polling-based)
├── ImportSummary.svelte         # Result card (inserted/updated/skipped/errors/duration)
├── HistoryDialog.svelte         # Import history list + detail view with rows + errors
├── TemplateDownloader.svelte    # Template download button
└── ReferenceValidator.svelte    # Reference dependency warnings

web/src/lib/stores/
└── import-export.svelte.ts      # Reactive store for import wizard state

web/src/lib/services/
└── import-export-service.ts     # API client for new endpoints
```

#### ImportWizard Steps

```
Step 1: DropZone → upload file (CSV or XLSX, drag-and-drop)
    │
    ▼
Step 2: ValidationSummary → show pipeline errors (if any), grouped by stage
    │
    ▼
Step 3: PreviewTable → show per-row status + diff + validation messages
    │
    ▼
Step 4: ProgressDialog → show progress during import execution
    │
    ▼
Step 5: ImportSummary → show result (inserted/updated/skipped/errors)
```

---

### Step 10: Module Migration + Cleanup

**Commit per module:** `refactor: migrate {module} to use import/export framework`

**Migration order** (dependency chain):

1. **Categories** — simplest, no references → reference implementation
2. **Brands** — no references → quick win
3. **Units of Measure** — no references → quick win
4. **Customers** — validates reference system
5. **Products** — most complex (3 reference types) → final validation

**Per-module migration:**

1. Schema already registered (Step 2)
2. Adapter already implemented (Step 8)
3. Update handler: replace old export/import with generic route registration
4. Remove old handler methods: `ExportXxx`, `ImportXxx`
5. Remove old service methods: `GetAllXxxForExport`, `ImportXxx`
6. Remove old repository methods: `GetAllXxxForExport`, `BulkUpsertXxx`
7. Remove old import row structs from domain
8. Update frontend page: replace `<ImportModal>` with `<ImportWizard>`

**Final cleanup commit:** `refactor: remove obsolete import/export code`

| File to Delete | Replaced By |
|---------------|-------------|
| `internal/importutil/parser.go` | `platform/importexport/import/parser.go` |
| `internal/shared/csv.go` | `platform/importexport/export/engine.go` |
| `web/src/shared/ui/ImportModal.svelte` | `lib/components/import-export/ImportWizard.svelte` |
| `web/src/shared/ui/ExportImportButtons.svelte` | `lib/components/import-export/BulkActionDropdown.svelte` |
| All `*ImportRow` structs | Generic `map[string]interface{}` parsing |
| All `GetAll*ForExport` methods | Adapter `ExportData()` |
| All `BulkUpsert*` methods | Adapter `Insert()` / `Update()` |

---

### Step 11: Performance Optimization

**Commit:** `perf: batch SQL operations and cache references`

1. **Reference cache**: `LoadReferences()` called once. Lookups use in-memory maps.
2. **Batch inserts**: `INSERT INTO ... VALUES (...), (...), ... ON CONFLICT ...`
3. **Batch updates**: `UPDATE ... FROM (VALUES ...)` or temp table + JOIN
4. **Transaction**: Single `pgx.Begin()` per import
5. **Stream parsing** (future): For 20k+ rows, stream CSV/XLSX instead of loading all

---

## 7. Interface Definitions

### Full Interface Set

```go
// internal/platform/importexport/interfaces.go

package importexport

import "context"

// Adapter is the thin bridge between the generic engine and a specific module.
// It handles ONLY domain-specific logic. Everything structural comes from ModuleSchema.
type Adapter interface {
    ModuleName() string

    // ValidateBusiness performs module-specific business validation.
    // Generic validation (type, reference, duplicate) is handled by the validation pipeline.
    ValidateBusiness(ctx context.Context, schema ModuleSchema, rows []map[string]interface{}) []ValidationError

    // MapToEntity converts a validated row to the domain entity for persistence.
    MapToEntity(ctx context.Context, schema ModuleSchema, row map[string]interface{}) (interface{}, error)

    // Repository provides data access operations.
    Repository() RepositoryActions
}

// RepositoryActions defines the persistence operations an adapter must provide.
type RepositoryActions interface {
    // Insert batch-inserts entities. Returns count of inserted rows.
    Insert(ctx context.Context, entities []interface{}) (int, error)

    // Update batch-updates entities. Returns count of updated rows.
    Update(ctx context.Context, entities []interface{}) (int, error)

    // ExportData fetches all data for export as []map[string]interface{}.
    ExportData(ctx context.Context, schema ModuleSchema) ([]map[string]interface{}, error)

    // LoadReferences loads reference data for validation.
    // Returns map[referenceColumn][]ReferenceItem.
    // Called once per import and cached by the engine.
    LoadReferences(ctx context.Context, schema ModuleSchema) (map[string][]ReferenceItem, error)
}

// ReferenceItem represents a single reference lookup value.
type ReferenceItem struct {
    Key   string      // e.g. brand name
    Value interface{} // e.g. brand ID or full struct
}

// SchemaRegistry stores ModuleSchema definitions.
type SchemaRegistry interface {
    Register(schema ModuleSchema) error
    Get(module string) (ModuleSchema, error)
    All() []ModuleSchema
}

// AdapterRegistry stores Adapter implementations.
type AdapterRegistry interface {
    Register(adapter Adapter) error
    Get(module string) (Adapter, error)
    Modules() []string
}

// Validator is a single validation stage in the pipeline.
type Validator interface {
    Name() string
    Validate(ctx context.Context, schema ModuleSchema, rows []map[string]interface{}, refs map[string][]ReferenceItem) []ValidationError
}
```

---

## 8. History Model

```
import_jobs
    │
    ├── id, module, schema_version, filename, status
    ├── total_rows, inserted, updated, skipped, error_count
    ├── progress_pct, started_at, completed_at, duration_ms
    ├── user_id, store_id, cancel_requested
    │
    ├── 1 ── import_snapshots (full row data at preview time)
    │          └── rows_data JSONB, schema_snapshot JSONB
    │
    ├── 1..* ── import_rows (per-row result)
    │             ├── row_number, status, entity_id
    │             ├── old_values JSONB, new_values JSONB
    │             └── changed_fields TEXT[]
    │
    └── 1..* ── import_errors (per-error detail)
                  ├── row_number, field, value
                  └── reason, suggestion, stage
```

This model supports future features:
- **Retry** — re-process rows where status = "error" using snapshot data
- **Rollback** — reverse changes using old_values
- **Diff** — compare two import jobs
- **Statistics** — aggregate by module, user, date range

---

## 9. Testing Strategy

### Layer 1: Schema Tests

- Schema validation (required fields, type consistency)
- Schema registry operations

### Layer 2: Validator Unit Tests

| Validator | What It Tests |
|-----------|---------------|
| TypeValidator | String/number/boolean/date parsing, boundary values |
| RequiredValidator | Missing required fields, empty strings |
| ReferenceValidator | Strict policy (fails on missing), AutoCreate (creates), Ignore (skips) |
| DuplicateValidator | BusinessKeys uniqueness across rows |
| Custom validator injection | Module adds custom validator, pipeline runs it |

### Layer 3: Engine Unit Tests (with mock adapter)

| Engine | What It Tests |
|--------|---------------|
| ImportEngine.Preview | Full preview lifecycle, error collection, snapshot creation |
| ImportEngine.Execute | Transaction commit/rollback, batch execution, history recording |
| ExportEngine | CSV + XLSX output structure, data type formatting |
| TemplateEngine | XLSX structure, reference sheets, data validation dropdowns |
| ProgressEngine | Status lifecycle, polling, cancellation flag |

### Layer 4: Adapter Integration Tests (with test DB)

| Adapter | What It Tests |
|---------|---------------|
| CategoryAdapter | Schema correctness, entity mapping, CRUD |
| BrandAdapter | Same |
| UOMAdapter | Same |
| CustomerAdapter | Same |
| ProductAdapter | Reference validation, business rules, entity mapping |

### Layer 5: API Integration Tests (with test DB)

- `POST /api/{module}/import/preview` — validation errors, preview data
- `POST /api/{module}/import` — successful import, rollback on error
- `GET /api/{module}/export?format=csv|xlsx` — valid file download
- `GET /api/{module}/template` — valid XLSX with reference sheets
- `GET /api/{module}/import/history` — paginated list
- `GET /api/{module}/import/history/{id}` — full detail with rows + errors
- `POST /api/{module}/import/{id}/cancel` — cancellation
- `GET /api/{module}/import/{id}/progress` — status polling

### Layer 6: Frontend Component Tests

- ImportWizard — all 5 steps, transitions, edge cases
- ValidationSummary — error display, empty state, grouped errors
- PreviewTable — insert/update/skip/error rows, diff rendering
- HistoryDialog — list, detail, pagination

---

## 10. Definition of Done

- [ ] Step 0: Brand and UOM modules extracted
- [ ] Step 1: Import history tables migrated
- [ ] Step 2: ModuleSchema + SchemaRegistry implemented
- [ ] Step 3: Pluggable Validator Pipeline with 6 default validators
- [ ] Step 4: Template Engine (schema → XLSX)
- [ ] Step 5: Import Engine (parse → validate → preview → execute → history)
- [ ] Step 6: Export Engine (CSV + XLSX from schema)
- [ ] Step 7: Progress Engine + cancellation support
- [ ] Step 8: Thin adapters for all 5 modules
- [ ] Step 9: All frontend components created and wired
- [ ] Step 10: All modules migrated, old code deleted
- [ ] Step 11: Batch SQL + reference cache optimized
- [ ] All unit tests pass
- [ ] All integration tests pass
- [ ] E2E tests pass
- [ ] UI is consistent across all 5 modules
- [ ] No switch/if-else branching on module type exists

---

## Commit Sequence

| # | Commit Message | Step |
|---|----------------|------|
| 1 | `refactor: extract Brand and UnitOfMeasure into independent modules` | 0 |
| 2 | `feat: add import history tables (jobs, snapshots, rows, errors)` | 1 |
| 3 | `feat: add ModuleSchema and SchemaRegistry` | 2 |
| 4 | `feat: add pluggable validation pipeline with 6 default validators` | 3 |
| 5 | `feat: add schema-driven template engine (ModuleSchema → XLSX)` | 4 |
| 6 | `feat: add schema-driven import engine (parse → validate → preview → execute)` | 5 |
| 7 | `feat: add schema-driven export engine (CSV + XLSX)` | 6 |
| 8 | `feat: add progress engine and import cancellation` | 7 |
| 9 | `feat: implement thin adapters for categories, brands, uoms, customers, products` | 8 |
| 10 | `feat: add reusable import/export frontend components` | 9 |
| 11 | `refactor: migrate categories to import/export framework` | 10 |
| 12 | `refactor: migrate brands to import/export framework` | 10 |
| 13 | `refactor: migrate uoms to import/export framework` | 10 |
| 14 | `refactor: migrate customers to import/export framework` | 10 |
| 15 | `refactor: migrate products to import/export framework` | 10 |
| 16 | `refactor: remove obsolete import/export code and import row structs` | 10 |
| 17 | `perf: batch SQL operations and cache references in import engine` | 11 |
