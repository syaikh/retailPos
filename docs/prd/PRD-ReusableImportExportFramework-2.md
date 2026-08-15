# Technical Design
# Reusable Import & Export Framework

---

# 1. Architecture Overview

The Import & Export Framework is designed as a reusable platform service rather than a feature owned by any specific module.

Instead of implementing import/export separately for Products, Brands, Categories, Customers, and future modules, the framework exposes a generic engine that delegates module-specific behavior to adapters.

This architecture follows the Open/Closed Principle.

The framework remains unchanged while new modules are added through adapters.

---

# 2. High Level Architecture

                         +----------------------+
                         |    Svelte Frontend   |
                         +----------+-----------+
                                    |
                                    |
                        Upload / Export Request
                                    |
                                    |
                  +-----------------v------------------+
                  |       Import / Export API          |
                  +-----------------+------------------+
                                    |
             +----------------------+----------------------+
             |                                             |
             |                                             |
+------------v------------+                   +------------v------------+
| Generic Import Engine   |                   | Generic Export Engine   |
+------------+------------+                   +------------+------------+
             |                                             |
             |                                             |
             +----------------------+----------------------+
                                    |
                                    |
                         ImportExportAdapter
                                    |
        +-------------+-------------+--------------+--------------+
        |             |                            |              |
        |             |                            |              |
 ProductAdapter  BrandAdapter          CategoryAdapter   CustomerAdapter

---

# 3. Design Principles

The framework must be

Reusable

Module agnostic

Testable

Extensible

Transaction safe

Strongly typed

Minimal duplication

Single workflow

---

# 4. Generic Engine Responsibilities

The engine knows

✔ how to read files

✔ how to validate templates

✔ how to parse rows

✔ how to create preview

✔ how to execute transaction

✔ how to generate reports

The engine DOES NOT know

what is Product

what is Brand

what is Customer

what is Supplier

That responsibility belongs to adapters.

---

# 5. Adapter Pattern

Every module implements the same interface.

Example

ProductAdapter

BrandAdapter

CategoryAdapter

CustomerAdapter

SupplierAdapter (future)

TaxAdapter (future)

WarehouseAdapter (future)

---

# 6. Import Adapter Interface

Each adapter is responsible for

Template Definition

Reference Loading

Reference Validation

Business Validation

Entity Mapping

Insert

Update

Export Mapping

The engine calls the adapter.

The adapter never calls the engine.

---

# 7. Responsibilities

Engine

Read Excel

Read CSV

Detect Template Version

Detect Required Sheets

Validate Required Columns

Parse Rows

Progress Tracking

Transaction

Summary

History

Error Report

Preview

--------------------------------

Adapter

Business Rules

Lookup Brand

Lookup Category

Lookup Unit

Duplicate SKU

Duplicate Barcode

Entity Conversion

Repository Calls

---

# 8. Folder Structure

backend/

modules/

products/

brands/

customers/

platform/

importexport/

engine/

adapter/

excel/

csv/

preview/

history/

report/

validation/

template/

interfaces/

frontend/

lib/

components/

import-export/

BulkActionDropdown

ImportWizard

PreviewTable

SummaryCard

HistoryDialog

TemplateDownloader

ReferenceValidator

stores/

services/

---

# 9. Import Lifecycle

Upload

↓

Parse Workbook

↓

Validate Template

↓

Load References

↓

Validate Rows

↓

Generate Preview

↓

Wait Confirmation

↓

Transaction

↓

History

↓

Summary

Every module follows this lifecycle.

---

# 10. Validation Pipeline

Validation executes in stages.

Stage 1

File Validation

↓

Stage 2

Workbook Validation

↓

Stage 3

Column Validation

↓

Stage 4

Type Validation

↓

Stage 5

Reference Validation

↓

Stage 6

Business Validation

↓

Stage 7

Duplicate Validation

↓

Preview

Errors stop progressing only for the affected row.

The framework should collect as many errors as possible.

---

# 11. Reference Cache

During validation

the engine loads every lookup once.

Example

Brand

Category

Unit

Customer Group

These are cached.

Rows should never perform repeated SQL queries.

Incorrect

5000 rows

↓

5000 Brand queries

Correct

Load all Brands once

↓

Memory Lookup

---

# 12. Preview Model

Each row has

Status

Insert

Update

Skip

Error

Changed Fields

Validation Messages

Old Entity

New Entity

This preview model is module independent.

---

# 13. History Model

Every import creates one Import Job.

Import Job

↓

contains

Import Items

↓

contains

Validation Errors

This enables future features

Retry

Rollback

Audit

Statistics

---

# 14. Transaction Strategy

Validation

↓

Preview

↓

User Confirmation

↓

Single Database Transaction

↓

Commit

If any fatal error occurs

Rollback entire import.

---

# 15. Export Engine

Export engine uses the same adapter.

Adapter defines

Columns

Order

Formatting

Reference Sheet

Dropdown Definitions

The engine generates

Excel

CSV

without knowing module details.

---

# 16. Future Extension

Adding a new module should require only

Create Adapter

Register Adapter

Done

No modification should be required inside

Engine

Preview

History

Validation

Template

Report

---

# 17. Module Registration

The framework should expose a registry.

Example

products

↓

ProductAdapter

brands

↓

BrandAdapter

customers

↓

CustomerAdapter

The engine resolves adapters through the registry.

No switch statement.

No if-else chain.

---

# 18. Error Handling

Framework errors

Template invalid

Workbook corrupt

Unsupported format

Permission denied

Adapter errors

Duplicate SKU

Brand not found

Category not found

Barcode exists

These remain separated.

---

# 19. Backend API

Each module exposes the same API shape.

POST

/import

POST

/import/preview

GET

/export

GET

/template

GET

/import/history

GET

/import/history/{id}

This keeps frontend reusable.

---

# 20. Frontend Components

Reusable components

BulkActionDropdown

ImportWizard

DropZone

ValidationSummary

PreviewTable

ProgressDialog

ImportSummary

ImportHistoryDialog

ReferenceWarning

These components should never contain Product-specific logic.

Everything is supplied by configuration.

---

# 21. Configuration Driven

Each adapter provides metadata.

Example

Columns

Primary Key

Lookup Fields

Editable Fields

Required Fields

Reference Sheets

Dropdown Columns

The engine uses metadata instead of hardcoded rules.

---

# 22. Testing Strategy

Generic Engine

Unit Tests

Adapter

Unit Tests

Repository

Integration Tests

Import API

Integration Tests

Frontend Wizard

Component Tests

End-to-End

Import Products

Import Brands

Import Customers

Export

Preview

History

The engine should have the highest test coverage because every module depends on it.
