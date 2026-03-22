package processor

import (
	"fmt"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	"github.com/aarondl/opt/null"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/services/conversion"
)

type (
	// EntityResolver resolves simulation-specific IDs or names to internal IDs.
	EntityResolver interface {
		ResolveTrack(simTrackID, simTrackName string) (trackID uint32, err error)
		ResolveDriver(simDriverID, simDriverName string) (driverID uint32, err error)
		ResolveCar(simCarID, simCarName string) (carID uint32, err error)
	}

	// Resolver transforms parsed input rows into resolved common result entries.
	Resolver struct {
		entityResolver EntityResolver
	}
	Result struct {
		TrackID  uint32
		Entries  []*models.ResultEntry
		Unmapped []*commonv1.UnresolvedMapping
	}
)

// NewResolver returns a Resolver using the provided entity resolver.
func NewResolver(entityResolver EntityResolver) *Resolver {
	return &Resolver{
		entityResolver: entityResolver,
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
		row := &inp.Results[i]
		entry := &models.ResultEntry{
			FinishingPosition: int32(row.FinPos),
			CompletedLaps:     int32(row.Laps),
			DriverName:        row.Name,
			CarName:           null.From(row.Car),
			Incidents:         null.From(int32(row.Incidents)),
			State:             conversion.ResultStateNormal,
			SourceRowNumber:   null.From(int32(i + 1)),
		}

		driverID, err := r.entityResolver.ResolveDriver(row.DriverID, row.Name)
		if err != nil {
			entry.State = conversion.ResultStateMappingError
			unresolved = append(unresolved, &commonv1.UnresolvedMapping{
				SourceValue: fmt.Sprintf("%s (name: %s)", row.DriverID, row.Name),
				MappingType: "driver",
			})
		} else {
			entry.DriverID = null.From(int32(driverID))
		}

		carID, err := r.entityResolver.ResolveCar(row.CarID, row.Car)
		if err != nil {
			entry.State = conversion.ResultStateMappingError
			unresolved = append(unresolved, &commonv1.UnresolvedMapping{
				SourceValue: fmt.Sprintf("%s (name: %s)", row.CarID, row.Car),
				MappingType: "car",
			})
		} else {
			entry.CarModelID = null.From(int32(carID))
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
		driverID, err := r.entityResolver.ResolveDriver("", entry.DriverName)
		if err != nil {
			entryHasUnresolved = true
			unresolved = append(unresolved, &commonv1.UnresolvedMapping{
				SourceValue: entry.DriverName,
				MappingType: "driver",
			})
		} else {
			entry.DriverID = null.From(int32(driverID))
		}

		carID, err := r.entityResolver.ResolveCar("", entry.CarName.GetOr(""))
		if err != nil {
			entryHasUnresolved = true
			unresolved = append(unresolved, &commonv1.UnresolvedMapping{
				SourceValue: entry.CarName.GetOr(""),
				MappingType: "car",
			})
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
