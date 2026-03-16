From the domain model in srlmgr/concept/docs, the models that should be represented as database tables and can exist without depending on other entities are:

1. RacingSimulation

- Independent: yes (no foreign key required).
- Why table: source-of-truth for supported platforms and import behavior.
- Suggested table: racing_sims.
- Key fields: id, name (unique), supported_import_formats, data_mapping, is_active, created_at.

2. Driver

- Independent: yes (no required FK in the model).
- Why table: core identity used by results, teams, and booking entries.
- Suggested table: drivers.
- Key fields: id, external_id (unique), name, simulation_ids, aliases, is_active, joined_at, last_imported_from.

3. PointSystem

- Independent: yes (referenced by seasons, but can be created standalone).
- Why table: reusable scoring configuration with lifecycle/state.
- Suggested table: point_systems.
- Key fields: id, name, description, position_points, is_active, created_at, updated_at.

Notes for this first pass:

1. These are the clean base tables to create first because they are upstream dependencies for most other models.
2. Some value objects attached to these could be separate tables later if querying/history is needed:

- NameChange for Driver (likely driver_name_changes).
- BonusRule and EligibilityRule for PointSystem (either normalized child tables or JSON columns).

If you want, next I can classify the remaining models into:

1. Dependent transactional tables (Series, Season, Event, Race, Result, Team, BookingEntry, ImportReview, PointSystemVersion).
2. Read-model/snapshot tables (Standing, TeamStanding, EventStanding).
3. Value objects that are better embedded vs normalized.
