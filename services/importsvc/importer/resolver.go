package importer

import (
	"fmt"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	"github.com/aarondl/opt/null"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/db/mytypes"
	"github.com/srlmgr/backend/log"
	"github.com/srlmgr/backend/services/conversion"
	"github.com/srlmgr/backend/services/importsvc/processor"
)

type (
	// EntityResolver resolves simulation-specific IDs or names to internal IDs.
	EntityResolver interface {
		ResolveTrack(simTrackID, simTrackName string) (trackID uint32, err error)
		ResolveDriver(simDriverID, simDriverName string) (driverID uint32, err error)
		ResolveCar(simCarID, simCarName string) (carID uint32, err error)
		ResolveTeam(ownDriverID uint32) (teamID uint32, err error)
		ResolveCarClass(ownCarID uint32) (carClassID uint32, err error)
	}

	// Resolver transforms parsed input rows into resolved common result entries.
	Resolver struct {
		entityResolver EntityResolver
		epi            *processor.EventProcInfo
	}
	Result struct {
		TrackID  uint32
		Entries  []*models.ResultEntry
		Unmapped []*commonv1.UnresolvedMapping
	}
)

// NewResolver returns a Resolver using the provided entity resolver.
//
//nolint:whitespace // editor/linter issue
func NewResolver(
	entityResolver EntityResolver,
	epi *processor.EventProcInfo,
) *Resolver {
	return &Resolver{
		entityResolver: entityResolver,
		epi:            epi,
	}
}

//nolint:funlen // function complexity is acceptable for row-by-row mapping
func (r *Resolver) ResolveInput(inp *ParsedImportPayload) (*Result, error) {
	if r.entityResolver == nil {
		return nil, fmt.Errorf("resolve input: entity resolver is not set")
	}

	entries := make([]*models.ResultEntry, len(inp.Results))
	unresolved := make([]*commonv1.UnresolvedMapping, 0)
	for i := range inp.Results {
		row := inp.Results[i]
		entry := &models.ResultEntry{
			FinishPosition: int32(row.FinPos),
			LapsCompleted:  int32(row.Laps),
			StartPosition:  null.From(int32(row.StartPos)),
			RawCarName:     null.From(row.Car),
			CarNumber:      null.From(row.CarNumber),
			Incidents:      null.From(int32(row.Incidents)),
			State:          conversion.ResultStateNormal,
		}
		if r.epi.Season.IsTeamBased {
			entry.RawTeamName = null.From(row.Name)
		} else {
			entry.RawDriverName = null.From(row.Name)
		}
		if row.QualiLapTime > 0 {
			entry.QualiLapTimeMS = null.From(int32(row.QualiLapTime))
		}
		if row.TotalTime > 0 {
			entry.TotalTimeMS = null.From(int32(row.TotalTime))
		}
		if row.FastestLapTime > 0 {
			entry.FastestLapTimeMS = null.From(int32(row.FastestLapTime))
		}
		//nolint:nestif // yes, it's complex
		if r.epi.Season.IsTeamBased {
			teamID, driverIDs, um := r.resolveTeam(row.TeamDrivers, row.Name)
			if um != nil {
				entry.State = conversion.ResultStateMappingError
				unresolved = append(unresolved, um)
			} else {
				entry.TeamID = null.From(int32(teamID))
				entry.TeamDrivers = null.From(mytypes.TeamDrivers{DriverIDs: driverIDs})
			}
		} else {
			driverID, um := r.resolveDriver(row.DriverID, row.Name)
			if um != nil {
				entry.State = conversion.ResultStateMappingError
				unresolved = append(unresolved, um)
			} else {
				entry.DriverID = null.From(int32(driverID))
			}
		}

		carID, um := r.resolveCar(row.CarID, row.Car)
		if um != nil {
			entry.State = conversion.ResultStateMappingError
			unresolved = append(unresolved, um)
		} else {
			entry.CarModelID = null.From(int32(carID))
		}

		// if multi-class, resolve car class as well
		if r.epi.Season.IsMulticlass {
			carClassID, um := r.resolveCarClass(carID, row.Car)
			if um != nil {
				entry.State = conversion.ResultStateMappingError
				unresolved = append(unresolved, um)
			} else {
				entry.CarClassID = null.From(int32(carClassID))
			}
		}
		entries[i] = entry
	}
	trackID, err := r.entityResolver.ResolveTrack(inp.Session.Track, inp.Session.Track)
	if err != nil {
		unresolved = append(unresolved, &commonv1.UnresolvedMapping{
			SourceValue: fmt.Sprintf("%s (name: %s)", inp.Session.Track, inp.Session.Track),
			MappingType: "track",
		})
	}
	return &Result{
		TrackID:  trackID,
		Entries:  entries,
		Unmapped: unresolved,
	}, nil
}

// returns
// - resolved resultEntries where input has mapping_error
// - still unresolved mappings
//
//nolint:whitespace,funlen // editor/linter issue
func (r *Resolver) ResolveNonMapped(
	inp *ParsedImportPayload,
	existing []*models.ResultEntry,
) (*Result, error) {
	if r.entityResolver == nil {
		return nil, fmt.Errorf("resolve input: entity resolver is not set")
	}
	resolvedResults := make([]*models.ResultEntry, 0)
	unresolved := make([]*commonv1.UnresolvedMapping, 0)
	for i := range existing {
		entry := existing[i]
		if entry.State != conversion.ResultStateMappingError {
			resolvedResults = append(resolvedResults, entry)
			continue
		}
		entryHasUnresolved := false
		driverID, um := r.resolveDriver("", entry.RawDriverName.GetOr(""))
		if um != nil {
			entryHasUnresolved = true
			unresolved = append(unresolved, um)
		} else {
			entry.DriverID = null.From(int32(driverID))
		}

		carID, um := r.resolveCar("", entry.RawCarName.GetOr(""))
		if um != nil {
			entryHasUnresolved = true
			unresolved = append(unresolved, um)
		} else {
			entry.CarModelID = null.From(int32(carID))
		}

		if entryHasUnresolved {
			entry.State = conversion.ResultStateMappingError
		} else {
			entry.State = conversion.ResultStateNormal
		}
		resolvedResults = append(resolvedResults, entry)
	}
	trackID, err := r.entityResolver.ResolveTrack(inp.Session.Track, inp.Session.Track)
	if err != nil {
		unresolved = append(unresolved, &commonv1.UnresolvedMapping{
			SourceValue: fmt.Sprintf("%s (name: %s)", inp.Session.Track, inp.Session.Track),
			MappingType: "track",
		})
	}

	return &Result{
		TrackID:  trackID,
		Entries:  resolvedResults,
		Unmapped: unresolved,
	}, nil
}

//nolint:whitespace // editor/linter issue
func (r *Resolver) resolveDriver(
	simDriverID, simDriverName string,
) (uint32, *commonv1.UnresolvedMapping) {
	driverID, err := r.entityResolver.ResolveDriver(simDriverID, simDriverName)
	if err != nil {
		sv := simDriverName
		if simDriverID != "" {
			sv = fmt.Sprintf("%s (name: %s)", simDriverID, simDriverName)
		}
		return 0, &commonv1.UnresolvedMapping{SourceValue: sv, MappingType: "driver"}
	}
	return driverID, nil
}

//nolint:whitespace // editor/linter issue
func (r *Resolver) resolveTeam(
	teamDrivers []*TeamDriver,
	rowName string,
) (uint32, []int32, *commonv1.UnresolvedMapping) {
	var teamDriverIDs []int32
	var dErr error
	for _, td := range teamDrivers {
		var driverID uint32
		driverID, dErr = r.entityResolver.ResolveDriver(td.DriverID, td.Name)
		if dErr == nil {
			teamDriverIDs = append(teamDriverIDs, int32(driverID))
			log.Debug("resolved team driver",
				log.Uint32("driverID", driverID),
				log.String("inputName", td.Name),
			)
		}

	}
	var teamID uint32
	for _, tdID := range teamDriverIDs {
		var resolveErr error
		teamID, resolveErr = r.entityResolver.ResolveTeam(uint32(tdID))
		if resolveErr == nil {
			break
		}
	}
	// TODO: we may need to enhance the UnresolvedMapping message
	// it should provided more infos about what is missing
	if teamID == 0 {
		return 0, nil, &commonv1.UnresolvedMapping{
			SourceValue: fmt.Sprintf("team with driver %d (name: %s)",
				teamDriverIDs, rowName),
			MappingType: "team",
		}
	}
	log.Debug("resolved team",
		log.Uint32("teamID", teamID),
		log.Any("driverIDs", teamDriverIDs),
		log.String("inputName", rowName),
	)
	return teamID, teamDriverIDs, nil
}

//nolint:whitespace // editor/linter issue
func (r *Resolver) resolveCar(
	simCarID,
	simCarName string,
) (uint32, *commonv1.UnresolvedMapping) {
	carID, err := r.entityResolver.ResolveCar(simCarID, simCarName)
	if err != nil {
		sv := simCarName
		if simCarID != "" {
			sv = fmt.Sprintf("%s (name: %s)", simCarID, simCarName)
		}
		return 0, &commonv1.UnresolvedMapping{SourceValue: sv, MappingType: "car"}
	}
	return carID, nil
}

//nolint:whitespace // editor/linter issue
func (r *Resolver) resolveCarClass(
	carID uint32,
	carName string,
) (uint32, *commonv1.UnresolvedMapping) {
	carClassID, err := r.entityResolver.ResolveCarClass(carID)
	if err != nil {
		return 0, &commonv1.UnresolvedMapping{
			SourceValue: fmt.Sprintf("car class for car %d (name: %s)", carID, carName),
			MappingType: "car_class",
		}
	}
	return carClassID, nil
}
