package importer

import (
	"errors"
	"fmt"
	"strings"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	"github.com/aarondl/opt/null"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/db/mytypes"
	"github.com/srlmgr/backend/grpc/services/conversion"
	"github.com/srlmgr/backend/grpc/services/importsvc/processor"
	"github.com/srlmgr/backend/repository"
)

type (
	// EntityResolver resolves simulation-specific IDs or names to internal IDs.
	// A missing mapping must wrap repository.ErrNotFound; other errors are fatal.
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
	seasonEntityValidator interface {
		ValidateSeasonCarModelVariant(carModelVariantID uint32) error
		ResolveSeasonDriver(driverID uint32) (*models.SeasonDriver, error)
		ResolveTeamCarNumber(teamID uint32) (string, error)
	}
	// Result contains resolved entries and their unresolved mapping diagnostics.
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
	return &Resolver{entityResolver: entityResolver, epi: epi}
}

// ResolveInput maps a parsed payload and marks entries with unresolved mappings.
func (r *Resolver) ResolveInput(inp *ParsedImportPayload) (*Result, error) {
	if err := r.validateInput(inp); err != nil {
		return nil, err
	}
	result := &Result{Entries: make([]*models.ResultEntry, len(inp.Results))}
	for i, row := range inp.Results {
		if row == nil {
			return nil, fmt.Errorf("resolve input: result row %d is nil", i)
		}
		entry := &models.ResultEntry{
			FinishPosition: int32(row.FinPos),
			LapsCompleted:  int32(row.Laps),
			StartPosition:  null.From(int32(row.StartPos)),
			RawCarName:     null.From(row.Car),
			CarNumber:      null.From(row.CarNumber),
			Incidents:      null.From(int32(row.Incidents)),
			Offtracks:      int32(row.Offtracks),
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
		mappings, err := r.resolveRow(row, entry)
		if err != nil {
			return nil, fmt.Errorf("resolve result row %d: %w", i, err)
		}
		r.addContext(mappings, entry, &i)
		result.Unmapped = append(result.Unmapped, mappings...)
		result.Entries[i] = entry
	}
	if err := r.resolveTrack(inp, result); err != nil {
		return nil, err
	}
	return result, nil
}

// ResolveNonMapped retries entries marked with mapping errors.
//
//nolint:whitespace // editor/linter issue
func (r *Resolver) ResolveNonMapped(
	inp *ParsedImportPayload,
	existing []*models.ResultEntry,
) (*Result, error) {
	if err := r.validateInput(inp); err != nil {
		return nil, err
	}
	result := &Result{Entries: make([]*models.ResultEntry, 0, len(existing))}
	for _, current := range existing {
		if current == nil {
			return nil, errors.New("resolve input: result entry is nil")
		}
		if current.State != conversion.ResultStateMappingError {
			result.Entries = append(result.Entries, current)
			continue
		}
		i, err := r.findInputRow(inp, current)
		if err != nil {
			return nil, fmt.Errorf("resolve result entry %d: %w", current.ID, err)
		}
		entry := *current
		mappings, err := r.resolveRow(inp.Results[i], &entry)
		if err != nil {
			return nil, fmt.Errorf("resolve result entry %d: %w", current.ID, err)
		}
		r.addContext(mappings, &entry, &i)
		result.Unmapped = append(result.Unmapped, mappings...)
		result.Entries = append(result.Entries, &entry)
	}
	if err := r.resolveTrack(inp, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *Resolver) validateInput(inp *ParsedImportPayload) error {
	if r.entityResolver == nil {
		return errors.New("resolve input: entity resolver is not set")
	}
	if inp == nil || r.epi == nil || r.epi.Season == nil {
		return errors.New("resolve input: payload or season is not set")
	}
	return nil
}

// findInputRow avoids pairing database results with imported rows by query order.
//
//nolint:whitespace // editor/linter issue
func (r *Resolver) findInputRow(
	inp *ParsedImportPayload,
	entry *models.ResultEntry,
) (int, error) {
	match := -1
	for i, row := range inp.Results {
		if row == nil || row.Car != entry.RawCarName.GetOr("") ||
			strings.TrimSpace(row.CarNumber) != strings.TrimSpace(entry.CarNumber.GetOr("")) {
			continue
		}
		name := entry.RawDriverName.GetOr("")
		if r.epi.Season.IsTeamBased {
			name = entry.RawTeamName.GetOr("")
		}
		if row.Name != name {
			continue
		}
		if match != -1 {
			return 0, errors.New("multiple imported rows match result entry")
		}
		match = i
	}
	if match == -1 {
		return 0, errors.New("no imported row matches result entry")
	}
	return match, nil
}

//nolint:funlen,gocyclo,nestif,whitespace // Independent mappings must all be checked.
func (r *Resolver) resolveRow(
	row *ResultRow,
	entry *models.ResultEntry,
) ([]*commonv1.UnresolvedMapping, error) {
	var mappings []*commonv1.UnresolvedMapping
	entry.DriverID = null.Val[int32]{}
	entry.TeamID = null.Val[int32]{}
	entry.TeamDrivers = null.Val[mytypes.TeamDrivers]{}
	entry.CarModelVariantID = null.Val[int32]{}
	entry.CarClassID = null.Val[int32]{}
	if r.epi.Season.IsTeamBased {
		teamID, driverIDs, teamMappings, err := r.resolveTeam(row.TeamDrivers, row.Name)
		if err != nil {
			return nil, err
		}
		mappings = append(mappings, teamMappings...)
		if teamID != 0 {
			entry.TeamID = null.From(int32(teamID))
			entry.TeamDrivers = null.From(mytypes.TeamDrivers{DriverIDs: driverIDs})
			numberMapping, err := r.validateTeamCarNumber(teamID, row.CarNumber)
			if err != nil {
				return nil, err
			}
			if numberMapping != nil {
				mappings = append(mappings, numberMapping)
			}
		}
	} else {
		driverID, seasonDriver, mapping, err := r.resolveDriver(row.DriverID, row.Name)
		if err != nil {
			return nil, err
		}
		if mapping != nil {
			mappings = append(mappings, mapping)
		}
		if driverID != 0 {
			entry.DriverID = null.From(int32(driverID))
			if mapping == nil {
				if numberMapping := r.validateDriverCarNumber(
					driverID,
					seasonDriver,
					row.CarNumber,
				); numberMapping != nil {
					mappings = append(mappings, numberMapping)
				}
			}
		}
	}

	carID, mapping, err := r.resolveCar(row.CarID, row.Car)
	if err != nil {
		return nil, err
	}
	if mapping != nil {
		mappings = append(mappings, mapping)
	}
	if carID != 0 {
		entry.CarModelVariantID = null.From(int32(carID))
	}
	if r.epi.Season.IsMulticlass && carID != 0 {
		classID, classMapping, err := r.resolveCarClass(carID, row.CarID, row.Car)
		if err != nil {
			return nil, err
		}
		if classMapping != nil {
			mappings = append(mappings, classMapping)
		} else {
			entry.CarClassID = null.From(int32(classID))
		}
	}
	entry.State = conversion.ResultStateNormal
	if len(mappings) > 0 {
		entry.State = conversion.ResultStateMappingError
	}
	return mappings, nil
}

func (r *Resolver) resolveTrack(inp *ParsedImportPayload, result *Result) error {
	id, err := r.entityResolver.ResolveTrack(inp.Session.Track, inp.Session.Track)
	if err != nil {
		if !errors.Is(err, repository.ErrNotFound) {
			return fmt.Errorf("resolve track: %w", err)
		}
		mapping := sourceNotFound(
			"track",
			fmt.Sprintf("%s (name: %s)", inp.Session.Track, inp.Session.Track),
			commonv1.SourceEntityType_SOURCE_ENTITY_TYPE_TRACK,
			"",
			inp.Session.Track,
		)
		r.addContext([]*commonv1.UnresolvedMapping{mapping}, nil, nil)
		result.Unmapped = append(result.Unmapped, mapping)
		return nil
	}
	if id == 0 {
		return errors.New("resolve track: resolver returned an empty id")
	}
	result.TrackID = id
	return nil
}

//nolint:whitespace // editor/linter issue
func (r *Resolver) addContext(
	mappings []*commonv1.UnresolvedMapping,
	entry *models.ResultEntry,
	index *int,
) {
	for _, mapping := range mappings {
		if r.epi.Series != nil && r.epi.Series.SimulationID > 0 {
			mapping.SimulationId = uint32Ptr(uint32(r.epi.Series.SimulationID))
		}
		if r.epi.Season.ID > 0 {
			mapping.SeasonId = uint32Ptr(uint32(r.epi.Season.ID))
		}
		if r.epi.Event != nil {
			if r.epi.Event.ID > 0 {
				mapping.EventId = uint32Ptr(uint32(r.epi.Event.ID))
			}
			if !r.epi.Event.EventDate.IsZero() {
				mapping.EventTime = timestamppb.New(r.epi.Event.EventDate)
			}
		}
		if entry != nil {
			if entry.ID > 0 {
				mapping.ResultEntryId = uint32Ptr(uint32(entry.ID))
			}
			if entry.RaceGridID > 0 {
				mapping.RaceGridId = uint32Ptr(uint32(entry.RaceGridID))
			}
		}
		if index != nil {
			mapping.ResultIndex = uint32Ptr(uint32(*index))
		}
	}
}

func uint32Ptr(value uint32) *uint32 { return &value }

func sourceEntity(id, name string) *commonv1.SourceEntity {
	source := &commonv1.SourceEntity{}
	if id != "" {
		source.Id = &id
	}
	if name != "" {
		source.Name = &name
	}
	return source
}

//nolint:whitespace // editor/linter issue
func sourceNotFound(
	kind, value string,
	entityType commonv1.SourceEntityType,
	id, name string,
) *commonv1.UnresolvedMapping {
	return &commonv1.UnresolvedMapping{
		SourceValue: value,
		MappingType: kind,
		Reason: &commonv1.UnresolvedMapping_SourceEntityNotFound{
			SourceEntityNotFound: &commonv1.SourceEntityNotFound{
				EntityType: entityType,
				Source:     sourceEntity(id, name),
			},
		},
	}
}

func sourceValue(id, name string) string {
	if id != "" {
		return fmt.Sprintf("%s (name: %s)", id, name)
	}
	return name
}

//nolint:nestif,whitespace // season validation follows entity resolution
func (r *Resolver) resolveDriver(
	id, name string,
) (uint32, *models.SeasonDriver, *commonv1.UnresolvedMapping, error) {
	driverID, err := r.entityResolver.ResolveDriver(id, name)
	if err != nil {
		if !errors.Is(err, repository.ErrNotFound) {
			return 0, nil, nil, fmt.Errorf("resolve driver: %w", err)
		}
		return 0, nil, sourceNotFound("driver", sourceValue(id, name),
			commonv1.SourceEntityType_SOURCE_ENTITY_TYPE_DRIVER, id, name), nil
	}
	if driverID == 0 {
		return 0, nil, nil, errors.New("resolve driver: resolver returned an empty id")
	}
	if validator, ok := r.entityResolver.(seasonEntityValidator); ok {
		seasonDriver, err := validator.ResolveSeasonDriver(driverID)
		if err != nil {
			if !errors.Is(err, repository.ErrNotFound) {
				return 0, nil, nil, fmt.Errorf("resolve season driver %d: %w", driverID, err)
			}
			return driverID, nil, &commonv1.UnresolvedMapping{
				SourceValue: fmt.Sprintf("%d (name: %s)", driverID, name), MappingType: "driver",
				Reason: &commonv1.UnresolvedMapping_SeasonDriverNotActive{
					SeasonDriverNotActive: &commonv1.SeasonDriverNotActive{
						DriverId:     driverID,
						SourceDriver: sourceEntity(id, name),
					},
				},
			}, nil
		}
		if seasonDriver == nil {
			return 0, nil, nil, fmt.Errorf(
				"resolve season driver %d: resolver returned nil",
				driverID,
			)
		}
		return driverID, seasonDriver, nil, nil
	}
	return driverID, nil, nil, nil
}

//nolint:funlen,whitespace // all drivers must be checked
func (r *Resolver) resolveTeam(
	drivers []*TeamDriver,
	name string,
) (uint32, []int32, []*commonv1.UnresolvedMapping, error) {
	if len(drivers) == 0 {
		return 0, nil, []*commonv1.UnresolvedMapping{
			{
				SourceValue: fmt.Sprintf("team with driver [] (name: %s)", name),
				MappingType: "team",
				Reason: &commonv1.UnresolvedMapping_TeamDriversMissing{
					TeamDriversMissing: &commonv1.TeamDriversMissing{SourceTeamName: name},
				},
			},
		}, nil
	}
	var ids []int32
	var checked []*commonv1.ResolvedTeamDriver
	var mappings []*commonv1.UnresolvedMapping
	for _, driver := range drivers {
		if driver == nil {
			return 0, nil, nil, errors.New("resolve team: imported driver is nil")
		}
		id, _, mapping, err := r.resolveDriver(driver.DriverID, driver.Name)
		if err != nil {
			return 0, nil, nil, err
		}
		if mapping != nil {
			mappings = append(mappings, mapping)
			continue
		}
		ids = append(ids, int32(id))
		checked = append(
			checked,
			&commonv1.ResolvedTeamDriver{
				DriverId:     id,
				SourceDriver: sourceEntity(driver.DriverID, driver.Name),
			},
		)
	}
	if len(mappings) > 0 {
		return 0, nil, mappings, nil
	}
	for _, id := range ids {
		teamID, err := r.entityResolver.ResolveTeam(uint32(id))
		if err == nil {
			if teamID == 0 {
				return 0, nil, nil, fmt.Errorf(
					"resolve team for driver %d: resolver returned an empty id",
					id,
				)
			}
			return teamID, ids, nil, nil
		}
		if !errors.Is(err, repository.ErrNotFound) {
			return 0, nil, nil, fmt.Errorf("resolve team for driver %d: %w", id, err)
		}
	}
	return 0, nil, []*commonv1.UnresolvedMapping{
		{
			SourceValue: fmt.Sprintf("team with driver %d (name: %s)", ids, name),
			MappingType: "team",
			Reason: &commonv1.UnresolvedMapping_TeamNotFound{
				TeamNotFound: &commonv1.TeamNotFound{SourceTeamName: name, DriversChecked: checked},
			},
		},
	}, nil
}

//nolint:whitespace // editor/linter issue
func (r *Resolver) resolveCar(
	id, name string,
) (uint32, *commonv1.UnresolvedMapping, error) {
	carID, err := r.entityResolver.ResolveCar(id, name)
	if err != nil {
		if !errors.Is(err, repository.ErrNotFound) {
			return 0, nil, fmt.Errorf("resolve car: %w", err)
		}
		return 0, sourceNotFound(
			"car",
			sourceValue(id, name),
			commonv1.SourceEntityType_SOURCE_ENTITY_TYPE_CAR,
			id,
			name,
		), nil
	}
	if carID == 0 {
		return 0, nil, errors.New("resolve car: resolver returned an empty id")
	}
	if validator, ok := r.entityResolver.(seasonEntityValidator); ok {
		if err := validator.ValidateSeasonCarModelVariant(carID); err != nil {
			if !errors.Is(err, repository.ErrNotFound) {
				return 0, nil, fmt.Errorf("validate season car %d: %w", carID, err)
			}
			return carID, &commonv1.UnresolvedMapping{
				SourceValue: fmt.Sprintf("%s (name: %s)", id, name), MappingType: "car",
				Reason: &commonv1.UnresolvedMapping_SeasonCarNotConfigured{
					SeasonCarNotConfigured: &commonv1.SeasonCarNotConfigured{
						CarModelVariantId: carID,
						SourceCar:         sourceEntity(id, name),
					},
				},
			}, nil
		}
	}
	return carID, nil, nil
}

//nolint:whitespace // editor/linter issue
func (r *Resolver) validateDriverCarNumber(
	driverID uint32,
	seasonDriver *models.SeasonDriver,
	imported string,
) *commonv1.UnresolvedMapping {
	if seasonDriver == nil || strings.TrimSpace(imported) == "" ||
		strings.TrimSpace(seasonDriver.CarNumber) == strings.TrimSpace(imported) {
		return nil
	}
	number := &commonv1.CarNumberMismatch{
		Owner:          &commonv1.CarNumberMismatch_DriverId{DriverId: driverID},
		ImportedNumber: imported, ConfiguredNumber: &seasonDriver.CarNumber,
	}
	return &commonv1.UnresolvedMapping{
		SourceValue: fmt.Sprintf(
			"car number %q for driver %d",
			strings.TrimSpace(imported),
			driverID,
		),
		MappingType: "car_number",
		Reason:      &commonv1.UnresolvedMapping_CarNumberMismatch{CarNumberMismatch: number},
	}
}

//nolint:whitespace // editor/linter issue
func (r *Resolver) validateTeamCarNumber(
	teamID uint32,
	imported string,
) (*commonv1.UnresolvedMapping, error) {
	if strings.TrimSpace(imported) == "" {
		return nil, nil
	}
	validator, ok := r.entityResolver.(seasonEntityValidator)
	if !ok {
		return nil, nil
	}
	configured, err := validator.ResolveTeamCarNumber(teamID)
	if err != nil {
		return nil, fmt.Errorf("resolve car number for team %d: %w", teamID, err)
	}
	if strings.TrimSpace(configured) == strings.TrimSpace(imported) {
		return nil, nil
	}
	number := &commonv1.CarNumberMismatch{
		Owner:          &commonv1.CarNumberMismatch_TeamId{TeamId: teamID},
		ImportedNumber: imported, ConfiguredNumber: &configured,
	}
	return &commonv1.UnresolvedMapping{
		SourceValue: fmt.Sprintf(
			"car number %q for team %d", strings.TrimSpace(imported), teamID,
		),
		MappingType: "car_number",
		Reason:      &commonv1.UnresolvedMapping_CarNumberMismatch{CarNumberMismatch: number},
	}, nil
}

//nolint:whitespace // editor/linter issue
func (r *Resolver) resolveCarClass(
	carID uint32,
	id, name string,
) (uint32, *commonv1.UnresolvedMapping, error) {
	classID, err := r.entityResolver.ResolveCarClass(carID)
	if err != nil {
		if !errors.Is(err, repository.ErrNotFound) {
			return 0, nil, fmt.Errorf("resolve car class for car %d: %w", carID, err)
		}
		return 0, &commonv1.UnresolvedMapping{
			SourceValue: fmt.Sprintf(
				"car class for car %d (name: %s)",
				carID,
				name,
			),
			MappingType: "car_class",
			Reason: &commonv1.UnresolvedMapping_CarClassNotConfigured{
				CarClassNotConfigured: &commonv1.CarClassNotConfigured{
					CarModelVariantId: carID,
					SourceCar:         sourceEntity(id, name),
				},
			},
		}, nil
	}
	if classID == 0 {
		return 0, nil, errors.New("resolve car class: resolver returned an empty id")
	}
	return classID, nil, nil
}
