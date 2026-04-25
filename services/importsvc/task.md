# Merge import parts

If the import allows multiple uploads, do the following:

From the merged zipPayload extract the race and quali entries. Start with the race entry and create ParsedImportPayload via ImportProcessor. Do this again with quali zip entry. If only one entry exists you don't need to merge, just continue with the storage

## Merging

The two ParsedImportPayload.Results need to be merged. Depending on the season settings isTeamBased this is either teamID (case true) or the driverID otherwise.
Merge the following from quali to race:

- quali.startPosition -> race.startPosition
- quali.QualiLapTime -> race.QualiLapTime

## Storage

This affects mainly the replaceResultEntriesForBatch method.

The combined ParsedImportPayload.Results need to be updated in the database via models.ResultEntry

- read existing entries first
- update an entry with the data from the combined Results
- persist the merged data (you may use flush and fill mechanic here)

Note: For imports that doesn't support mutliple uploads just use a flush&fill to store the result entries
