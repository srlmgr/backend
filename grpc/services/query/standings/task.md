Tasks for ComputeStandingsInput

- the caller defines ReferenceID to return either the DriverID or TeamID. This is defined upfront

- group Bookings by eventID
- start with empty data
- for each eventID in eventIDs:
    - collect the raceGridIDs for the eventID from the grouped bookings
    - sum points by referenceID using all sourceTypes. This is also referred as rawPoints
    - sum bonus_points by referenceID. Use sourceType "qualification_pos", "least_incidents", "fastest_lap", "top_n_finishers"
    - sum penalty_points by referenceID. Use sourceType penalty_points
    - create StandingData as copy of the previous standing entry. if none exits, init with 0 values
    - set prev_position to the position of the referenceID in previous run, otherwise 0
    - increase num_events by one if there is an entry Participations by ReferenceID for one of the raceGridIDs for the current eventID
    - increase num_races by one for each entry in Participations by ReferenceID for one of the raceGridIDs for the current eventID
    - increase num_wins by one for each entry in Participations by ReferenceID for one of the raceGridIDs for the current eventID where FinishPosition = 1
    - increase num_podiums by one for each entry in Participations by ReferenceID for one of the raceGridIDs for the current eventID where FinishPosition <= 3
    - increase num_top5 by one for each entry in Participations by ReferenceID for one of the raceGridIDs for the current eventID where FinishPosition <= 5
    - increase num_top10 by one for each entry in Participations by ReferenceID for one of the raceGridIDs for the current eventID where FinishPosition <= 10

The SkipMode means the following:

- it affects only the total_points attribute
- SkipModeNever: do not handle skip events at all
- SkipModeAlways: The points are calculated by the raw points of all entries up to and including the current eventID. The worst NumSkip entries are skipped and not taken into computation. If there a less than NumSkip events, the points should be set to 0.
- SkipModeIfApplicable: only apply if len(EventIDs) >= (NumTotalEvents-NumSkip).

the return values are the Standing entries for the last eventID of eventIDs and the eventIDs of the skipped events.

for testing use as few data as possible. also consider the following cases

- a referenceID does not show up in one eventID
- an event has two races (raceGridIDs) and a referenceID is only in one of them

Do not care about linter warnings.
