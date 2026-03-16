---
name: db-table-planning
description: "Plan relational database tables from a domain model. Use for schema design, table classification, foreign-key dependency mapping, value object embedding vs normalization, and rollout ordering for transactional and read-model tables."
argument-hint: "Describe the domain model scope and constraints (auditability, query patterns, migration limits)."
---

# DB Table Planning

## What This Skill Produces

- A table-by-table plan grouped by dependency level.
- Clear classification into independent base tables, dependent transactional tables, and read-model or snapshot tables.
- A decision log for value objects: embed first or normalize now.
- A pragmatic implementation order that reduces migration risk.

## When to Use

Use this when you need to turn a domain model into practical relational tables, especially when balancing clean normalization with delivery speed.

Trigger phrases:

- db schema plan
- table design from domain model
- classify independent and dependent tables
- read model table planning
- embed vs normalize decision

## Inputs to Gather First

1. Domain entities and key relationships.
2. Required auditability and history requirements.
3. High-priority query patterns.
4. Migration and rollout constraints.

## Procedure

1. Identify independent base entities.
   Reasoning:

- Find entities that can exist without foreign keys to other domain entities.
  Output:
- Candidate base tables with core fields and uniqueness constraints.

2. Classify remaining entities as dependent transactional tables.
   Reasoning:

- Keep source-of-truth business events and state transitions in normalized transactional tables.
  Output:
- Each table with required foreign keys, key columns, and important uniqueness rules.

3. Classify derived views as read-model or snapshot tables.
   Reasoning:

- Separate rebuildable, denormalized query tables from transactional truth.
  Output:
- Snapshot tables with rebuild strategy and refresh trigger assumptions.

4. Decide value object storage using embed-first default.
   Branching logic:

- Embed as JSON or arrays when structure is tightly coupled to parent and rarely queried independently.
- Normalize into child tables when independent lifecycle, audit trail, validation, reuse, or frequent filtering is required.
  Output:
- Explicit embed vs normalize decisions for each value object.

5. Build dependency-aware rollout order.
   Recommended sequence:

- Independent base tables.
- Core hierarchy tables (for example: series then seasons then events then races).
- Participation and registration tables.
- Outcome tables (for example: results).
- Versioning or policy history tables.
- Read-model or snapshot tables last.
  Output:
- Ordered migration plan with prerequisites.

6. Add integrity and operability checks.
   Include:

- Primary key and foreign key strategy.
- Uniqueness constraints and conflict handling.
- Soft-delete or active-state policy.
- Rebuild path for all read models.
  Output:
- Checklist of constraints and operational safeguards.

## Quality Criteria

A complete result should:

- Show each table with classification and rationale.
- List dependencies explicitly for every dependent table.
- Distinguish transactional truth from rebuildable projections.
- Justify each normalize-now decision with a concrete query, lifecycle, or audit requirement.
- Provide an implementation order that avoids orphaned foreign keys.
- Include practical indexing hints for common join and lookup paths.

## Output Template

Use this structure in responses:

1. Independent base tables

- table name
- reason
- key fields

2. Dependent transactional tables

- table name
- depends on
- key fields
- constraints
- indexing hints

3. Read-model or snapshot tables

- table name
- depends on
- rebuild source

4. Value object strategy

- object name
- embed or normalize
- reason

5. Migration order

- ordered steps

6. Validation checklist

- integrity, lifecycle, and rebuild checks
