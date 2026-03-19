package authn

// anonymousProcedures holds procedures that allow unauthenticated access.
var anonymousProcedures = map[string]bool{
	// Query service read operations are publicly accessible.
	"/backend.query.v1.QueryService/GetDriverStandings":     true,
	"/backend.query.v1.QueryService/GetTeamStandings":       true,
	"/backend.query.v1.QueryService/GetEventResults":        true,
	"/backend.query.v1.QueryService/GetEventBookingEntries": true,
	"/backend.query.v1.QueryService/ListSimulations":        true,
	"/backend.query.v1.QueryService/ListSeries":             true,
	"/backend.query.v1.QueryService/ListSeasons":            true,
	"/backend.query.v1.QueryService/ListEvents":             true,
	"/backend.query.v1.QueryService/ListDrivers":            true,
	"/backend.query.v1.QueryService/ListTeams":              true,
	"/backend.query.v1.QueryService/ListPointSystems":       true,
	"/backend.query.v1.QueryService/ListTracks":             true,
	"/backend.query.v1.QueryService/ListTrackLayouts":       true,
	"/backend.query.v1.QueryService/ListCarManufacturers":   true,
}

// IsAnonymousProcedure returns true if the given RPC procedure allows
// unauthenticated access.
func IsAnonymousProcedure(procedure string) bool {
	return anonymousProcedures[procedure]
}
