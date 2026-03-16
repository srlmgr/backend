From the domain model in srlmgr/concept/docs, the remaining models can be classified into dependent transactional tables, read-model or snapshot tables, and value objects that are better embedded vs normalized.

1. Series

- Class: dependent transactional table.
- Why table: top-level competition container that groups seasons and defines the long-lived league structure.
- Suggested table: series.
- Depends on: no hard parent entity in the listed model set, but becomes an upstream dependency for seasons.
- Key fields: id, name, slug, description, simulation_id, is_active, created_at, updated_at.
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

4. Race

- Class: dependent transactional table.
- Why table: stores each competitive session inside an event and anchors result rows.
- Suggested table: races.
- Depends on: events.
- Key fields: id, event_id, name, session_type, order_index, laps, duration_seconds, status, started_at, ended_at.
- Notes: use order_index when an event contains practice, qualifying, sprint, and feature sessions.

5. Result

- Class: dependent transactional table.
- Why table: durable record of a driver's classified outcome in a race, required for standings and auditability.
- Suggested table: results.
- Depends on: races, drivers, optionally teams.
- Key fields: id, race_id, driver_id, team_id, finishing_position, classified_position, grid_position, laps_completed, total_time_ms, best_lap_ms, status, points_awarded, imported_at.
- Notes: enforce one result per race_id plus driver_id unless the domain explicitly allows duplicates for penalties or reclassifications.

6. Team

- Class: dependent transactional table.
- Why table: represents a managed competition unit that can participate across seasons and aggregate results.
- Suggested table: teams.
- Depends on: commonly season-scoped or series-scoped depending on the domain rule.
- Key fields: id, season_id, name, external_id, is_active, created_at, updated_at.
- Notes: if teams persist across seasons, split identity from membership later instead of overloading a single table.

7. BookingEntry

- Class: dependent transactional table.
- Why table: explicit registration or entry state for a driver participating in a season, event, or team context.
- Suggested table: booking_entries.
- Depends on: drivers, and usually seasons or events; optionally teams.
- Key fields: id, driver_id, season_id, event_id, team_id, car_number, status, registered_at, confirmed_at, notes.
- Notes: choose either season-level or event-level ownership first; support both only if the model clearly requires it.

8. ImportReview

- Class: dependent transactional table.
- Why table: preserves operator review state for imported race data and supports traceability for corrections.
- Suggested table: import_reviews.
- Depends on: racing_sims, and usually the imported target entity such as result sets, events, or races.
- Key fields: id, simulation_id, source_reference, target_type, target_id, status, review_notes, reviewed_by, reviewed_at, created_at.
- Notes: target_type plus target_id is a pragmatic polymorphic reference if imports can land on multiple aggregate types.

9. PointSystemVersion

- Class: dependent transactional table.
- Why table: tracks scoring-rule evolution without mutating historical results or season configurations.
- Suggested table: point_system_versions.
- Depends on: point_systems.
- Key fields: id, point_system_id, version_number, rules_blob, effective_from, effective_to, created_at.
- Notes: use this when scoring changes must remain auditable; otherwise the base point_systems table may be enough initially.

Read-model or snapshot tables

1. Standing

- Class: read-model or snapshot table.
- Why table: materialized season-level driver standings for fast reads and historical snapshots.
- Suggested table: standings.
- Depends on: seasons, drivers.
- Key fields: id, season_id, driver_id, rank, points, wins, podiums, penalties, snapshot_version, calculated_at.
- Notes: can be rebuilt from results, so treat as derived data rather than primary source of truth.

2. TeamStanding

- Class: read-model or snapshot table.
- Why table: materialized team championship table for leaderboard reads and exports.
- Suggested table: team_standings.
- Depends on: seasons, teams.
- Key fields: id, season_id, team_id, rank, points, wins, podiums, penalties, snapshot_version, calculated_at.
- Notes: same derived-data rule as standings.

3. EventStanding

- Class: read-model or snapshot table.
- Why table: event-scoped summary view for weekend rankings, aggregate points, or combined session outcomes.
- Suggested table: event_standings.
- Depends on: events, drivers, optionally teams.
- Key fields: id, event_id, driver_id, team_id, rank, points, aggregate_time_ms, snapshot_version, calculated_at.
- Notes: useful if event ranking combines multiple races or sessions.

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
