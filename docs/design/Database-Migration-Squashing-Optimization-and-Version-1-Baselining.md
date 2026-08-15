You are a **database migration expert**. Your task is to analyze the repository's existing SQL migration history and safely **squash, optimize, and baseline all migrations into a single clean Version 1 migration**, while preserving the exact effective database state required by the application.

The goal is to replace a potentially large history of messy, overlapping, redundant, or superseded migrations with a clean baseline that can initialize a new database correctly from scratch.

## Core Objective

The repository currently contains existing SQL migration files.

You must:

1. Inspect and understand the complete existing migration history.
2. Create a completely empty temporary database.
3. Run **all existing migrations from scratch**, in their normal order.
4. Treat the resulting database as the authoritative representation of the current migration state.
5. Extract the final schema.
6. Identify and preserve intentional initial/reference data.
7. Create a single clean **Version 1 baseline migration**.
8. Remove the old migration history and replace it with the new baseline.
9. Prove that the new baseline produces an equivalent database state to the complete legacy migration chain.
10. Run relevant tests and verify application compatibility.

This is primarily a **migration-history cleanup/baselining task**, not a database redesign.

Do not change the application's intended schema or behavior merely to make the migration cleaner.

---

# Phase 1 — Inspect the Migration System

Before modifying anything, inspect the repository thoroughly.

Identify:

- database engine and version
- migration framework/tool
- migration directory
- all migration files
- migration naming/versioning convention
- migration execution order
- migration configuration
- migration scripts
- Makefile targets
- Docker/Compose configuration
- CI configuration
- test database setup
- application database initialization/bootstrap process

Determine exactly how the application normally executes migrations.

Also inspect the migration files for:

- tables
- columns
- indexes
- primary keys
- foreign keys
- unique constraints
- check constraints
- defaults
- sequences
- identity columns
- enums/custom types
- schemas/namespaces
- extensions
- views
- materialized views
- triggers
- functions/procedures
- `INSERT`
- `UPDATE`
- `DELETE`
- `UPSERT` / `ON CONFLICT`
- data backfills
- seed/reference data
- migration bookkeeping

Also search the application code for assumptions about:

- migration versions
- migration-specific objects
- seeded IDs
- default/reference records
- database functions
- views
- enums
- constraints
- special indexes

**Do not modify the repository during this phase.**

---

# Phase 2 — Preserve the Original Migration History

Before making destructive changes, ensure the original migration history can be recovered.

Prefer using Git or a temporary worktree/branch.

Do not delete the original migration files until:

- the legacy database has been successfully created
- the baseline has been generated
- equivalence testing has passed

The original migration chain is the reference implementation for this task.

---

# Phase 3 — Build the Legacy Database

Create a **completely isolated temporary database** for this task.

Requirements:

- It must start completely empty.
- Do not use production.
- Do not use staging.
- Do not reset an existing development database.
- Do not modify any persistent database that belongs to the project.
- Use temporary credentials/configuration where appropriate.

Then run **every existing migration from the beginning**, using the same migration mechanism and order used by the application.

Do not skip migrations merely because they appear obsolete.

Do not manually rewrite migrations just because they look messy.

If a migration fails:

1. Stop.
2. Investigate the failure.
3. Determine whether the failure is caused by the migration itself, environment configuration, or an existing dependency.
4. Do not silently work around the failure.
5. Document the problem.

The database produced by the complete legacy migration chain is the primary source of truth.

Record the commands used so the process is reproducible.

---

# Phase 4 — Validate the Legacy Database

Once all legacy migrations have successfully executed, inspect the resulting database.

Verify the final state of:

### Schema

- schemas/namespaces
- tables
- columns
- data types
- nullability
- default values
- primary keys
- foreign keys
- unique constraints
- check constraints
- indexes
- sequences
- identity configuration
- enums/custom types
- extensions
- views
- materialized views
- triggers
- functions/procedures

Pay particular attention to objects that were:

- created and later modified
- renamed
- dropped
- recreated
- replaced
- duplicated
- temporarily introduced during migration

The baseline must represent the **final effective state**, not the historical sequence of changes.

---

# Phase 5 — Generate the Version 1 Baseline

The new Version 1 migration must contain both:

1. the final database schema
2. intentional initial/reference data

Do not treat this as a schema-only task.

## 5.1 — Generate the Final Schema

Use the database engine's native schema-dump tooling, such as:

- `pg_dump`
- `mysqldump`
- or the appropriate equivalent

Generate the schema from the fully migrated temporary database.

The generated schema must contain everything required to recreate the final database structure from an empty database.

Preserve:

- tables
- columns
- data types
- defaults
- constraints
- indexes
- sequences
- identity definitions
- enums/custom types
- extensions
- views
- materialized views
- triggers
- functions/procedures
- schemas/namespaces

Exclude migration-framework bookkeeping tables unless the migration framework explicitly requires them as part of the baseline.

---

## 5.2 — Handle Existing Initial and Reference Data

The existing migrations may contain intentional initial data in addition to schema changes.

Do not assume that a schema dump will preserve this data.

Analyze all migrations for:

- `INSERT`
- `UPDATE`
- `DELETE`
- `UPSERT`
- `ON CONFLICT`
- data migrations
- backfills
- seed data
- reference/master data
- default system records

Classify the data.

### A. Required Initial / Reference Data

Preserve data that is intentionally required for a newly initialized database.

Examples may include:

- default roles
- permissions
- payment methods
- product categories
- units of measure
- tax configuration
- system configuration
- default statuses
- required lookup/reference records
- other application-defined defaults

This data must be preserved in Version 1.

### B. Historical Migration Data

Do not automatically preserve data that existed only to facilitate a historical schema transition.

Examples:

- temporary migration markers
- transitional values
- temporary backfill records
- records that were subsequently deleted
- data used only during an intermediate migration

Only preserve such data if it is still intentionally part of the final application state.

### C. Development/Test/Sample Data

Do not include:

- test fixtures
- sample data
- development-only records
- temporary debugging data
- environment-specific data

unless the project explicitly defines them as required baseline data.

---

## 5.3 — Derive the Final Intended Data State

The final state of the legacy database is the starting point for determining baseline data, but do not blindly dump all rows.

For every data-changing migration:

1. Determine what the migration was intended to accomplish.
2. Determine whether the resulting records are required in a fresh production database.
3. Determine whether later migrations modify or remove those records.
4. Preserve only the final intended state.

For example:

```text
Migration 001:
INSERT payment_method "Cash"

Migration 005:
INSERT payment_method "Credit Card"

Migration 012:
UPDATE "Credit Card" → "Card"

Migration 020:
INSERT payment_method "QRIS"
```

The baseline should contain:

```text
Cash
Card
QRIS
```

It should not reproduce the historical sequence of operations.

If reference/master data has stable IDs that application code relies on, preserve those IDs.

Prefer deterministic explicit `INSERT` statements for baseline reference data.

---

# Phase 6 — Clean and Normalize the Baseline

Review the generated migration rather than blindly committing the raw database dump.

The Version 1 migration should be:

- deterministic
- reproducible
- readable
- logically ordered
- minimal
- free of historical migration artifacts
- free of temporary objects
- free of redundant statements
- compatible with the existing migration framework

Do not optimize the actual database design unless explicitly required.

In particular, do not remove or alter objects merely because they appear unused.

Do not arbitrarily:

- remove columns
- remove indexes
- remove constraints
- change data types
- change nullability
- change defaults
- rename objects
- alter foreign-key behavior
- change IDs
- change reference-data semantics

The objective is to eliminate **migration history complexity**, not application/database functionality.

Also inspect the generated SQL for:

- usernames
- passwords
- credentials
- hostnames
- environment-specific values
- machine-specific paths
- temporary database names
- timestamps that should not be fixed
- unrelated data
- migration bookkeeping artifacts

Do not commit secrets.

---

# Phase 7 — Create the New Version 1 Migration

Using the project's existing migration naming convention, create a single new baseline migration.

The migration must be executable against a completely empty database.

It must establish:

```text
Empty database
      ↓
Version 1 baseline
      ↓
Final schema
      +
Required initial/reference data
      ↓
Database ready for the application
```

Ensure the migration framework recognizes the new baseline correctly.

Do not leave the old migrations active alongside Version 1 unless the migration framework explicitly requires a special transition mechanism.

---

# Phase 8 — Prove Schema Equivalence

This is a mandatory validation step.

Create two separate temporary databases.

## Database A — Legacy

Initialize from an empty database using the complete original migration chain.

## Database B — Baseline

Initialize from an empty database using only the new Version 1 migration.

Then compare the resulting schemas.

Compare at minimum:

- schemas
- tables
- columns
- data types
- nullability
- defaults
- primary keys
- foreign keys
- unique constraints
- check constraints
- indexes
- sequences
- identity definitions
- enums/custom types
- extensions
- views
- materialized views
- triggers
- functions/procedures

Use native database comparison tools where available.

Supplement with normalized schema dumps if necessary.

Every difference must be investigated.

Do not assume that a difference is harmless.

The desired result is:

> **Zero unintended schema differences.**

If an intentional difference exists, document:

- what differs
- why it differs
- why it is safe
- why it does not change application behavior

---

# Phase 9 — Prove Initial/Reference Data Equivalence

Schema equivalence alone is insufficient.

For every table identified as containing required baseline/reference data, compare:

```text
Database A — legacy migrations
vs.
Database B — Version 1 baseline
```

Verify equivalent final contents.

Examples include:

- roles
- permissions
- payment methods
- categories
- UOMs
- system configuration
- tax configuration
- status/reference tables
- other required lookup/master data

Prefer deterministic comparisons based on stable business keys or primary keys.

If generated values differ, such as:

- timestamps
- generated UUIDs
- auto-increment IDs

determine whether those values are semantically important.

If application code depends on stable IDs, preserve the original IDs.

The desired result is:

> **The baseline database contains the same intentional initial/reference data as the legacy database.**

Do not compare or require equality for transactional data that should not exist in a fresh database.

---

# Phase 10 — Fresh Database Initialization Test

Destroy the temporary baseline database and create another completely empty database.

Run only the new Version 1 migration.

Verify that:

- migration succeeds
- all required objects are created
- all required reference data exists
- application startup/bootstrap succeeds
- relevant database tests succeed

This proves that the new baseline is independently capable of initializing a fresh database.

---

# Phase 11 — Application Compatibility Testing

Run the project's relevant tests after replacing the migration history.

At minimum, run:

- database tests
- integration tests
- migration tests
- relevant backend tests
- application startup/bootstrap tests

If practical, run the broader test suite.

Also verify important application functionality that depends directly on:

- seeded data
- reference data
- database constraints
- views
- functions
- triggers
- enums
- indexes

Do not declare the migration squash complete if the baseline cannot initialize and support the application correctly.

---

# Phase 12 — Replace the Old Migration Files

Only after the validation above succeeds:

- remove the old migration files
- keep the new Version 1 migration
- preserve the migration directory structure
- update migration configuration only if necessary

Do not modify unrelated application code.

If the project uses Git, inspect the diff carefully before considering the task complete.

---

# Phase 13 — Final Git Diff Review

Review all changes.

Confirm:

- old migrations were intentionally removed
- one new Version 1 baseline exists
- no unrelated application files changed
- no accidental configuration changes occurred
- no credentials/secrets were committed
- no development/test data was included
- no transactional data was included
- no temporary migration artifacts remain
- no migration bookkeeping artifacts were accidentally included

Pay particular attention to the SQL dump because database dump tools may include environment-specific information that should not be committed.

---

# Safety Rules

These rules are mandatory:

1. **Never touch production.**
2. **Never touch staging.**
3. Do not reset an existing development database.
4. Use isolated temporary databases.
5. Preserve the original migration files until validation succeeds.
6. Do not silently skip a migration.
7. Do not silently resolve migration failures.
8. Do not redesign the schema.
9. Do not remove objects simply because they appear unused.
10. Do not remove required initial/reference data.
11. Do not include transactional/business data in the baseline.
12. Do not include test/development data unless explicitly intended as baseline data.
13. Do not commit credentials or environment-specific configuration.
14. Do not claim equivalence without actually comparing the two databases.
15. Do not consider the task complete until a completely empty database can be initialized using only Version 1.

---

# Final Deliverables

At the end, provide a concise report containing:

## 1. Migration Summary

- database engine/version
- migration framework
- number of original migrations
- number of migrations after squashing
- new Version 1 migration name
- total migration history reduction

## 2. Schema Summary

List the important objects preserved:

- tables
- indexes
- constraints
- sequences
- enums/types
- views
- materialized views
- triggers
- functions/procedures
- extensions

## 3. Initial/Reference Data Summary

List the categories of baseline data preserved.

For example:

- payment methods
- roles
- permissions
- categories
- UOMs
- system settings
- tax configuration

Also state whether any historical data was intentionally excluded and why.

## 4. Validation Results

Report:

```text
Legacy migration from empty DB:       PASS / FAIL
Baseline migration from empty DB:    PASS / FAIL
Schema equivalence:                   PASS / FAIL
Reference data equivalence:           PASS / FAIL
Application compatibility:            PASS / FAIL
Relevant tests:                       PASS / FAIL
```

## 5. Differences

If any differences exist between the legacy and baseline databases, list every one and explain whether it is:

- intentional
- harmless
- application-impacting
- unresolved

Do not hide differences.

## 6. Tests Executed

List the exact commands used and their results.

## 7. Files Changed

List:

- deleted migration files
- new Version 1 migration
- any other modified files

## 8. Remaining Risks

Identify anything that could not be verified automatically or requires human review.

---

# Definition of Done

The task is complete only when all of the following are true:

```text
Original migrations
       ↓
Empty Database A
       ↓
Final Schema
+
Final Required Reference Data
       │
       │
       │  compare
       ▼
Empty Database B
       ↑
       │
New Version 1
       ↓
Final Schema
+
Final Required Reference Data
```

And:

- Database A initializes successfully.
- Database B initializes successfully.
- Schema differences are zero or explicitly justified.
- Required initial/reference data is equivalent.
- No unintended historical migration artifacts remain.
- No transactional data has been incorporated.
- No secrets have been included.
- Application compatibility tests pass.
- The new Version 1 migration can initialize a completely empty database.
- The Git diff contains only intentional changes.

**Do not stop after generating the SQL dump. The actual success criterion is that the new Version 1 baseline faithfully replaces the complete legacy migration history while preserving the final schema and all intentional initial/reference data.**