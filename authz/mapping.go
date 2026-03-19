package authz

import "fmt"

const commandServiceName = "backend.command.v1.CommandService"

type scopeKind string

const (
	scopeNone       scopeKind = "none"
	scopeSeries     scopeKind = "series"
	scopeSimulation scopeKind = "simulation"
)

// ProcedurePolicy describes authorization requirements for one RPC procedure.
type ProcedurePolicy struct {
	Capability     string
	AllowAnonymous bool
	Scope          scopeKind
}

func procedure(service, method string) string {
	return fmt.Sprintf("/%s/%s", service, method)
}

func defaultProcedurePolicies() map[string]ProcedurePolicy {
	policies := map[string]ProcedurePolicy{}
	addQueryPolicies(policies)
	addCommandPolicies(policies)
	addImportPolicies(policies)
	addAdminPolicies(policies)
	return policies
}

func addQueryPolicies(policies map[string]ProcedurePolicy) {
	queryService := "backend.query.v1.QueryService"
	for _, method := range []string{
		"GetDriverStandings",
		"GetTeamStandings",
		"GetEventResults",
		"GetEventBookingEntries",
		"ListSimulations",
		"ListSeries",
		"ListSeasons",
		"ListEvents",
		"ListDrivers",
		"ListTeams",
		"ListPointSystems",
		"ListTracks",
		"ListTrackLayouts",
		"ListCarManufacturers",
		"ListCarModels",
	} {
		policies[procedure(queryService, method)] = ProcedurePolicy{
			Capability:     "query.read",
			AllowAnonymous: true,
			Scope:          scopeNone,
		}
	}
}

func addCommandPolicies(policies map[string]ProcedurePolicy) {
	addSeasonWriteCommandPolicies(policies)
	addSimulationScopedCommandPolicies(policies)
	addMasterDataCommandPolicies(policies)
}

func addSeasonWriteCommandPolicies(policies map[string]ProcedurePolicy) {
	commandService := commandServiceName
	for _, method := range []string{
		"CreateSeason",
		"CreateEvent",
		"CreateRace",
		"CreateDriver",
		"UpsertDriverSimulationIdentity",
		"CreateTeam",
		"AddDriverToTeam",
		"CreatePointSystem",
		"UpdatePointSystem",
	} {
		policies[procedure(commandService, method)] = ProcedurePolicy{
			Capability: "season.write",
			Scope:      scopeSeries,
		}
	}
}

func addSimulationScopedCommandPolicies(policies map[string]ProcedurePolicy) {
	commandService := commandServiceName
	for _, method := range []string{
		"CreateSimulation",
		"UpsertSimulationTrackLayoutAlias",
		"UpsertSimulationCarAlias",
	} {
		policies[procedure(commandService, method)] = ProcedurePolicy{
			Capability: "simulation.write",
			Scope:      scopeSimulation,
		}
	}

	for _, method := range []string{
		"CreateSeries",
	} {
		policies[procedure(commandService, method)] = ProcedurePolicy{
			Capability: "series.write",
			Scope:      scopeSimulation,
		}
	}
}

func addMasterDataCommandPolicies(policies map[string]ProcedurePolicy) {
	commandService := commandServiceName
	for _, method := range []string{
		"CreateTrack",
		"CreateTrackLayout",
		"CreateCarManufacturer",
		"CreateCarModel",
	} {
		policies[procedure(commandService, method)] = ProcedurePolicy{
			Capability: "master_data.write",
			Scope:      scopeNone,
		}
	}
}

func addImportPolicies(policies map[string]ProcedurePolicy) {
	importService := "backend.import.v1.ImportService"
	for _, method := range []string{
		"UploadResultsFile",
		"GetPreprocessPreview",
		"ApplyResultEdits",
		"ComputeDriverBookingEntries",
		"ComputeTeamBookingEntries",
	} {
		policies[procedure(importService, method)] = ProcedurePolicy{
			Capability: "import.write",
			Scope:      scopeSeries,
		}
	}
	policies[procedure(importService, "FinalizeEventProcessing")] = ProcedurePolicy{
		Capability: "import.finalize",
		Scope:      scopeSeries,
	}
}

func addAdminPolicies(policies map[string]ProcedurePolicy) {
	adminService := "backend.admin.v1.AdminService"
	for _, method := range []string{
		"MarkResultState",
		"UpdateBookingEntryPoints",
		"CreateManualBookingEntry",
	} {
		policies[procedure(adminService, method)] = ProcedurePolicy{
			Capability: "admin.write",
			Scope:      scopeSeries,
		}
	}
}
