From the domain model in srlmgr/concept/docs, the remaining models can be classified into dependent transactional tables, read-model or snapshot tables, and value objects that are better embedded vs normalized.

1. Series

- Class: dependent transactional table.
- Why table: top-level competition container that groups seasons and defines the long-lived league structure.
- Suggested table: series.
- Depends on: no hard parent entity in the listed model set, but becomes an upstream dependency for seasons.
- Key fields: id, name, description, simulation_id, is_active, created_at, updated_at.
- Notes: treat simulation_id as the first meaningful foreign key if the series is tied to a RacingSimulation.

2. Season

- Class: dependent transactional table.
- Why table: represents a bounded competition cycle with standings, events, and rules attached.
- Suggested table: seasons.
- Depends on: series, point_systems.
- Key fields: id, series_id, point_system_id, name, short_name, starts_at, ends_at, status, is_active, created_at, updated_at.
- Notes: season is the main aggregation point for events, standings, and versioned scoring references.

3. Event

- Class: dependent transactional table.
- Why table: captures a scheduled meeting within a season and acts as the parent for races and event-level standings.
- Suggested table: events.
- Depends on: seasons.
- Key fields: id, season_id, name, round_number, venue, starts_at, ends_at, status, created_at, updated_at.
- Notes: unique constraints often make sense on season_id plus round_number.

6. Team

- Class: dependent transactional table.
- Why table: represents a managed competition unit that can participate across seasons and aggregate results.
- Suggested table: teams.
- Depends on: commonly season-scoped or series-scoped depending on the domain rule.
- Key fields: id, season_id, name, external_id, is_active, created_at, updated_at.
- Notes: if teams persist across seasons, split identity from membership later instead of overloading a single table.

Value objects better embedded vs normalized

1. Embed first

- RacingSimulation.supported_import_formats: embed as JSON or array unless formats need independent lifecycle or cross-simulation analytics.
- RacingSimulation.data_mapping: embed as JSON because it behaves like configuration tied to the sim definition.
- Driver.aliases: embed as JSON or text array unless alias lookup becomes a primary query path.
- PointSystem.position_points: embed as JSON because it is tightly coupled scoring configuration.
- PointSystemVersion.rules_blob: embed as JSON when version payloads are treated atomically.
- ImportReview.review_notes or import metadata: embed as JSON if structure varies by source.

2. Normalize when history, querying, or reuse matters

- Driver NameChange: normalize into driver_name_changes when rename history or effective date queries are required.
- PointSystem BonusRule: normalize when bonus rules need independent validation, ordering, or analytics.
- PointSystem EligibilityRule: normalize when eligibility logic must be audited or selectively queried.
- Team membership history: normalize if drivers can join, leave, or represent multiple teams over time.
- BookingEntry state transitions: normalize into a history table if operational auditing matters.
- Import review decisions or field-level overrides: normalize when reviewers need a durable audit trail per changed field.

Implementation order for the next pass

1. Create series and seasons immediately after the independent base tables.
2. Add events and races next so competitive structure exists before ingesting outcomes.
3. Add teams and booking_entries once participant registration rules are clear.
4. Add results only after race, driver, and optional team relationships are settled.
5. Add point_system_versions if scoring changes must be historically preserved from day one.
6. Add standings tables last as derived read models, not as core transactional storage.

Practical modeling rule

1. Keep transactional tables normalized around identities and foreign keys.
2. Keep read models denormalized and rebuildable.
3. Keep value objects embedded until a concrete query, audit, or lifecycle requirement justifies another table.
