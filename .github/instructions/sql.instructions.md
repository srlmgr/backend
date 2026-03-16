---
description: "Instructions for writing SQL code "
applyTo: "**/*.sql"
---

# SQL Code Style Instructions

## Output location policy

- Store migrations in db/migrate/migrations
- Use 3-digit index based filenames
- Keep one concern per migration

## Tool policy

- create statements for PostgreSQL
- use github.com/golang-migrate/migrate/v4 as migration tool
- use transactions in migration

## Schema policy

- use snake_case, plural tables
- use id as primary key name
- use serial type for primary keys
- use timestamp with time zone for date columns, such as created_at and updated_at
- use foreign keys for relationships
- use indexes on foreign keys and frequently queried columns
- all entity represententing tables should have created_at and updated_at columns along with created_by and updated_by columns for auditability
- add constraints via alter statements in the same migration after creating the table to avoid circular dependency issues during migration
