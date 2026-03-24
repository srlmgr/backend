package conversion

// Import format strings as persisted in the database.
const (
	ImportFormatJSON = "json"
	ImportFormatCSV  = "csv"
)

// Race session type strings as persisted in the database.
const (
	RaceSessionTypeQualifying = "qualifying"
	RaceSessionTypeHeat       = "heat"
	RaceSessionTypeRace       = "race"
)

// Event status strings as persisted in the database.
const (
	EventStatusScheduled = "scheduled"
	EventStatusCompleted = "completed"
	EventStatusCancelled = "canceled"
)

// Event processing state strings as persisted in the database.
const (
	EventProcessingStateDraft                 = "draft"
	EventProcessingStateRawImported           = "raw_imported"
	EventProcessingStateMappingError          = "mapping_error"
	EventProcessingStatePreprocessed          = "pre_processed"
	EventProcessingStateDriverEntriesComputed = "driver_entries_computed"
	EventProcessingStateTeamEntriesComputed   = "team_entries_computed"
	EventProcessingStateComputed              = "computed"
	EventProcessingStateFinalized             = "finalized"
)

// Result state strings as persisted in the database.
const (
	ResultStateNormal       = "normal"
	ResultStateDQ           = "dq"
	ResultStateMappingError = "mapping_error"
)
