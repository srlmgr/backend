package authn

// Principal is the normalized authenticated identity used throughout the backend.
type Principal struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Tenant        string   `json:"tenant"`
	Roles         []string `json:"roles"`
	SimulationIDs []string `json:"simulationIDs"`
	SeriesIDs     []string `json:"seriesIDs"`
	Source        string   `json:"source"`
}
