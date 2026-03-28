// Package repository provides an in-memory repository implementation for tests.
//
//nolint:lll,dupl,funlen // test setups
package repository

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/aarondl/opt/null"
	"github.com/gofrs/uuid/v5"
	"github.com/lib/pq"

	"github.com/srlmgr/backend/db/models"
	mytypes "github.com/srlmgr/backend/db/mytypes"
	rootrepo "github.com/srlmgr/backend/repository"
	"github.com/srlmgr/backend/repository/bookingentries"
	"github.com/srlmgr/backend/repository/cars"
	"github.com/srlmgr/backend/repository/drivers"
	"github.com/srlmgr/backend/repository/eventprocessingaudit"
	"github.com/srlmgr/backend/repository/events"
	"github.com/srlmgr/backend/repository/importbatches"
	"github.com/srlmgr/backend/repository/pointsystems"
	"github.com/srlmgr/backend/repository/races"
	"github.com/srlmgr/backend/repository/racingsims"
	"github.com/srlmgr/backend/repository/repoerrors"
	"github.com/srlmgr/backend/repository/resultentries"
	"github.com/srlmgr/backend/repository/seasons"
	"github.com/srlmgr/backend/repository/series"
	"github.com/srlmgr/backend/repository/standings"
	"github.com/srlmgr/backend/repository/teams"
	"github.com/srlmgr/backend/repository/tracks"
)

type mapEntityRepo[M any, S any] struct {
	mu     sync.RWMutex
	nextID int32
	data   map[int32]*M
	getID  func(*M) int32
	setID  func(*M, int32)
	apply  func(*M, *S)
}

//nolint:whitespace // editor/linter issue
func newMapEntityRepo[M, S any](
	getID func(*M) int32,
	setID func(*M, int32),
	apply func(*M, *S),
	initial ...*M,
) *mapEntityRepo[M, S] {
	repo := &mapEntityRepo[M, S]{
		nextID: 1,
		data:   make(map[int32]*M, len(initial)),
		getID:  getID,
		setID:  setID,
		apply:  apply,
	}

	for _, entity := range initial {
		cloned := cloneModel(entity)
		id := repo.getID(cloned)
		if id == 0 {
			id = repo.nextID
			repo.setID(cloned, id)
		}
		if id >= repo.nextID {
			repo.nextID = id + 1
		}
		repo.data[id] = cloned
	}

	return repo
}

func (r *mapEntityRepo[M, S]) LoadByID(_ context.Context, id int32) (*M, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entity, ok := r.data[id]
	if !ok {
		return nil, fmt.Errorf("entity %d: %w", id, repoerrors.ErrNotFound)
	}

	return cloneModel(entity), nil
}

func (r *mapEntityRepo[M, S]) LoadAll(_ context.Context) ([]*M, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]int32, 0, len(r.data))
	for id := range r.data {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	items := make([]*M, 0, len(ids))
	for _, id := range ids {
		items = append(items, cloneModel(r.data[id]))
	}

	return items, nil
}

func (r *mapEntityRepo[M, S]) DeleteByID(_ context.Context, id int32) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.data, id)
	return nil
}

func (r *mapEntityRepo[M, S]) Create(_ context.Context, input *S) (*M, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	entity := new(M)
	if input != nil {
		r.apply(entity, input)
	}

	id := r.getID(entity)
	if id == 0 {
		id = r.nextID
		r.setID(entity, id)
	}
	if id >= r.nextID {
		r.nextID = id + 1
	}
	if _, exists := r.data[id]; exists {
		return nil, fmt.Errorf("entity %d already exists", id)
	}

	stored := cloneModel(entity)
	r.data[id] = stored

	return cloneModel(stored), nil
}

func (r *mapEntityRepo[M, S]) Update(_ context.Context, id int32, input *S) (*M, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	entity, ok := r.data[id]
	if !ok {
		return nil, fmt.Errorf("entity %d: %w", id, repoerrors.ErrNotFound)
	}

	updated := cloneModel(entity)
	if input != nil {
		r.apply(updated, input)
	}
	r.setID(updated, id)
	r.data[id] = updated

	return cloneModel(updated), nil
}

func cloneModel[M any](entity *M) *M {
	if entity == nil {
		return nil
	}
	cloned := *entity
	return &cloned
}

type racingSimsEntityRepo struct {
	*mapEntityRepo[models.RacingSim, models.RacingSimSetter]
}
type pointSystemsEntityRepo struct {
	*mapEntityRepo[models.PointSystem, models.PointSystemSetter]
}
type pointRulesEntityRepo struct {
	*mapEntityRepo[models.PointRule, models.PointRuleSetter]
}
type driversEntityRepo struct {
	*mapEntityRepo[models.Driver, models.DriverSetter]
}

//nolint:whitespace // editor/linter issue
func (r *driversEntityRepo) FindByName(
	ctx context.Context, arg string,
) (*models.Driver, error) {
	items, err := r.LoadAll(ctx)
	if err != nil {
		return nil, err
	}

	for _, item := range items {
		if item == nil || item.Name != arg {
			continue
		}
		return item, nil
	}

	return nil, fmt.Errorf(
		"driver with name %q: %w",
		arg,
		repoerrors.ErrNotFound,
	)
}

type simulationDriverAliasesEntityRepo struct {
	*mapEntityRepo[models.SimulationDriverAlias, models.SimulationDriverAliasSetter]
}

//nolint:whitespace // multiline signature style
func (r *simulationDriverAliasesEntityRepo) FindBySimID(
	ctx context.Context,
	simID int32,
	aliases ...string,
) (*models.SimulationDriverAlias, error) {
	items, err := r.LoadBySimulationID(ctx, simID)
	if err != nil {
		return nil, err
	}

	for _, item := range items {
		if item == nil || !slices.Contains(aliases, item.SimulationDriverID) {
			continue
		}
		return item, nil
	}

	return nil, fmt.Errorf(
		"simulation driver alias %q for simulation %d: %w",
		aliases,
		simID,
		repoerrors.ErrNotFound,
	)
}

type tracksEntityRepo struct {
	*mapEntityRepo[models.Track, models.TrackSetter]
}
type trackLayoutsEntityRepo struct {
	*mapEntityRepo[models.TrackLayout, models.TrackLayoutSetter]
}

//nolint:whitespace // multiline signature style
func (r *trackLayoutsEntityRepo) LoadByTrackID(
	ctx context.Context,
	trackID int32,
) ([]*models.TrackLayout, error) {
	items, err := r.LoadAll(ctx)
	if err != nil {
		return nil, err
	}

	filtered := make([]*models.TrackLayout, 0, len(items))
	for _, item := range items {
		if item == nil || item.TrackID != trackID {
			continue
		}
		filtered = append(filtered, item)
	}

	return filtered, nil
}

type simulationTrackLayoutAliasesEntityRepo struct {
	*mapEntityRepo[models.SimulationTrackLayoutAlias, models.SimulationTrackLayoutAliasSetter]
}

//nolint:whitespace // multiline signature style
func (r *simulationTrackLayoutAliasesEntityRepo) LoadBySimulationID(
	ctx context.Context,
	simulationID int32,
) ([]*models.SimulationTrackLayoutAlias, error) {
	items, err := r.LoadAll(ctx)
	if err != nil {
		return nil, err
	}

	filtered := make([]*models.SimulationTrackLayoutAlias, 0, len(items))
	for _, item := range items {
		if item == nil || item.SimulationID != simulationID {
			continue
		}
		filtered = append(filtered, item)
	}

	return filtered, nil
}

//nolint:whitespace // multiline signature style
func (r *simulationTrackLayoutAliasesEntityRepo) FindBySimID(
	ctx context.Context,
	simID int32,
	aliases ...string,
) (*models.SimulationTrackLayoutAlias, error) {
	items, err := r.LoadAll(ctx)
	if err != nil {
		return nil, err
	}

	for _, item := range items {
		if item == nil || item.SimulationID != simID ||
			!slices.Contains(aliases, item.ExternalName) {
			continue
		}
		return item, nil
	}

	return nil, fmt.Errorf(
		"simulation track layout alias %q for simulation %d: %w",
		aliases,
		simID,
		repoerrors.ErrNotFound,
	)
}

type carManufacturersEntityRepo struct {
	*mapEntityRepo[models.CarManufacturer, models.CarManufacturerSetter]
}
type carBrandsEntityRepo struct {
	*mapEntityRepo[models.CarBrand, models.CarBrandSetter]
}
type carModelsEntityRepo struct {
	*mapEntityRepo[models.CarModel, models.CarModelSetter]
	brands *carBrandsEntityRepo
}
type simulationCarAliasesEntityRepo struct {
	*mapEntityRepo[models.SimulationCarAlias, models.SimulationCarAliasSetter]
}

//nolint:whitespace // multiline signature style
func (r *simulationCarAliasesEntityRepo) LoadBySimulationID(
	ctx context.Context,
	simulationID int32,
) ([]*models.SimulationCarAlias, error) {
	items, err := r.LoadAll(ctx)
	if err != nil {
		return nil, err
	}

	filtered := make([]*models.SimulationCarAlias, 0, len(items))
	for _, item := range items {
		if item == nil || item.SimulationID != simulationID {
			continue
		}
		filtered = append(filtered, item)
	}

	return filtered, nil
}

//nolint:whitespace // multiline signature style
func (r *simulationCarAliasesEntityRepo) FindBySimID(
	ctx context.Context,
	simID int32,
	aliases ...string,
) (*models.SimulationCarAlias, error) {
	items, err := r.LoadAll(ctx)
	if err != nil {
		return nil, err
	}

	for _, item := range items {
		if item == nil || item.SimulationID != simID ||
			!slices.Contains(aliases, item.ExternalName) {
			continue
		}
		return item, nil
	}

	return nil, fmt.Errorf(
		"simulation car alias %q for simulation %d: %w",
		aliases,
		simID,
		repoerrors.ErrNotFound,
	)
}

type seriesEntityRepo struct {
	*mapEntityRepo[models.Series, models.SeriesSetter]
}

//nolint:whitespace // multiline signature style
func (r *seriesEntityRepo) LoadBySimulationID(
	ctx context.Context,
	simulationID int32,
) ([]*models.Series, error) {
	items, err := r.LoadAll(ctx)
	if err != nil {
		return nil, err
	}

	filtered := make([]*models.Series, 0, len(items))
	for _, item := range items {
		if item == nil || item.SimulationID != simulationID {
			continue
		}
		filtered = append(filtered, item)
	}

	return filtered, nil
}

//nolint:whitespace // multiline signature style
func (r *simulationDriverAliasesEntityRepo) LoadBySimulationID(
	ctx context.Context,
	simulationID int32,
) ([]*models.SimulationDriverAlias, error) {
	items, err := r.LoadAll(ctx)
	if err != nil {
		return nil, err
	}

	filtered := make([]*models.SimulationDriverAlias, 0, len(items))
	for _, item := range items {
		if item == nil || item.SimulationID != simulationID {
			continue
		}
		filtered = append(filtered, item)
	}

	return filtered, nil
}

//nolint:whitespace // multiline signature style
func (r *carBrandsEntityRepo) LoadByManufacturerID(
	ctx context.Context,
	manufacturerID int32,
) ([]*models.CarBrand, error) {
	items, err := r.LoadAll(ctx)
	if err != nil {
		return nil, err
	}

	filtered := make([]*models.CarBrand, 0, len(items))
	for _, item := range items {
		if item == nil || item.ManufacturerID != manufacturerID {
			continue
		}
		filtered = append(filtered, item)
	}

	return filtered, nil
}

//nolint:whitespace // multiline signature style
func (r *carModelsEntityRepo) LoadByManufacturerID(
	ctx context.Context,
	manufacturerID int32,
) ([]*models.CarModel, error) {
	brands, err := r.brands.LoadByManufacturerID(ctx, manufacturerID)
	if err != nil {
		return nil, err
	}

	brandIDs := make(map[int32]bool, len(brands))
	for _, b := range brands {
		if b != nil {
			brandIDs[b.ID] = true
		}
	}

	allModels, err := r.LoadAll(ctx)
	if err != nil {
		return nil, err
	}

	filtered := make([]*models.CarModel, 0, len(allModels))
	for _, item := range allModels {
		if item == nil || !brandIDs[item.BrandID] {
			continue
		}
		filtered = append(filtered, item)
	}

	return filtered, nil
}

type seasonsEntityRepo struct {
	*mapEntityRepo[models.Season, models.SeasonSetter]
}

//nolint:whitespace // multiline signature style
func (r *seasonsEntityRepo) LoadBySeriesID(
	ctx context.Context,
	seriesID int32,
) ([]*models.Season, error) {
	items, err := r.LoadAll(ctx)
	if err != nil {
		return nil, err
	}

	filtered := make([]*models.Season, 0, len(items))
	for _, item := range items {
		if item == nil || item.SeriesID != seriesID {
			continue
		}
		filtered = append(filtered, item)
	}

	return filtered, nil
}

type eventsEntityRepo struct {
	*mapEntityRepo[models.Event, models.EventSetter]
}

//nolint:whitespace // multiline signature style
func (r *eventsEntityRepo) LoadBySeasonID(
	ctx context.Context,
	seasonID int32,
) ([]*models.Event, error) {
	items, err := r.LoadAll(ctx)
	if err != nil {
		return nil, err
	}

	filtered := make([]*models.Event, 0, len(items))
	for _, item := range items {
		if item == nil || item.SeasonID != seasonID {
			continue
		}
		filtered = append(filtered, item)
	}

	return filtered, nil
}

type racesEntityRepo struct {
	*mapEntityRepo[models.Race, models.RaceSetter]
}

//nolint:whitespace // multiline signature style
func (r *racesEntityRepo) LoadByEventID(
	ctx context.Context,
	eventID int32,
) ([]*models.Race, error) {
	items, err := r.LoadAll(ctx)
	if err != nil {
		return nil, err
	}

	filtered := make([]*models.Race, 0, len(items))
	for _, item := range items {
		if item == nil || item.EventID != eventID {
			continue
		}
		filtered = append(filtered, item)
	}

	return filtered, nil
}

type raceGridsEntityRepo struct {
	*mapEntityRepo[models.RaceGrid, models.RaceGridSetter]
}

//nolint:whitespace // multiline signature style
func (r *raceGridsEntityRepo) LoadByRaceID(
	ctx context.Context,
	raceID int32,
) ([]*models.RaceGrid, error) {
	items, err := r.LoadAll(ctx)
	if err != nil {
		return nil, err
	}

	filtered := make([]*models.RaceGrid, 0, len(items))
	for _, item := range items {
		if item == nil || item.RaceID != raceID {
			continue
		}
		filtered = append(filtered, item)
	}

	return filtered, nil
}

type teamsEntityRepo struct {
	*mapEntityRepo[models.Team, models.TeamSetter]
}

//nolint:whitespace // multiline signature style
func (r *teamsEntityRepo) LoadBySeasonID(
	ctx context.Context,
	seasonID int32,
) ([]*models.Team, error) {
	items, err := r.LoadAll(ctx)
	if err != nil {
		return nil, err
	}

	filtered := make([]*models.Team, 0, len(items))
	for _, item := range items {
		if item == nil || item.SeasonID != seasonID {
			continue
		}
		filtered = append(filtered, item)
	}

	return filtered, nil
}

type teamDriversEntityRepo struct {
	*mapEntityRepo[models.TeamDriver, models.TeamDriverSetter]
}

//nolint:whitespace // multiline signature style
func (r *teamDriversEntityRepo) LoadByTeamID(
	ctx context.Context,
	teamID int32,
) ([]*models.TeamDriver, error) {
	items, err := r.LoadAll(ctx)
	if err != nil {
		return nil, err
	}

	filtered := make([]*models.TeamDriver, 0, len(items))
	for _, item := range items {
		if item == nil || item.TeamID != teamID {
			continue
		}
		filtered = append(filtered, item)
	}

	return filtered, nil
}

type importBatchesEntityRepo struct {
	*mapEntityRepo[models.ImportBatch, models.ImportBatchSetter]
}

//nolint:whitespace // multiline signature style
func (r *importBatchesEntityRepo) LoadByRaceID(
	ctx context.Context,
	raceID int32,
) (*models.ImportBatch, error) {
	items, err := r.LoadAll(ctx)
	if err != nil {
		return nil, err
	}

	for _, item := range items {
		if item == nil || item.RaceID != raceID {
			continue
		}
		return item, nil
	}

	return nil, fmt.Errorf("import batch for race %d: %w", raceID, repoerrors.ErrNotFound)
}

//nolint:whitespace // multiline signature style
func (r *importBatchesEntityRepo) DeleteByRaceID(
	ctx context.Context,
	raceID int32,
) error {
	items, err := r.LoadAll(ctx)
	if err != nil {
		return err
	}

	for _, item := range items {
		if item == nil || item.RaceID != raceID {
			continue
		}
		if err := r.DeleteByID(ctx, item.ID); err != nil {
			return err
		}
	}

	return nil
}

type resultEntriesEntityRepo struct {
	*mapEntityRepo[models.ResultEntry, models.ResultEntrySetter]
}

//nolint:whitespace // multiline signature style
func (r *resultEntriesEntityRepo) CreateMany(
	ctx context.Context,
	input []*models.ResultEntrySetter,
) ([]*models.ResultEntry, error) {
	if len(input) == 0 {
		return []*models.ResultEntry{}, nil
	}

	return nil, fmt.Errorf("not implemented")
}

//nolint:whitespace // multiline signature style
func (r *resultEntriesEntityRepo) LoadByRaceID(
	ctx context.Context,
	raceID int32,
) ([]*models.ResultEntry, error) {
	items, err := r.LoadAll(ctx)
	if err != nil {
		return nil, err
	}

	filtered := make([]*models.ResultEntry, 0, len(items))
	for _, item := range items {
		if item == nil || item.RaceID != raceID {
			continue
		}
		filtered = append(filtered, item)
	}

	return filtered, nil
}

//nolint:whitespace // multiline signature style
func (r *resultEntriesEntityRepo) LoadByRaceGridID(
	ctx context.Context,
	raceGridID int32,
) ([]*models.ResultEntry, error) {
	items, err := r.LoadAll(ctx)
	if err != nil {
		return nil, err
	}

	filtered := make([]*models.ResultEntry, 0, len(items))
	for _, item := range items {
		if item == nil || item.RaceGridID != raceGridID {
			continue
		}
		filtered = append(filtered, item)
	}

	return filtered, nil
}

//nolint:whitespace // multiline signature style
func (r *resultEntriesEntityRepo) LoadByState(
	ctx context.Context,
	state string,
) ([]*models.ResultEntry, error) {
	items, err := r.LoadAll(ctx)
	if err != nil {
		return nil, err
	}

	filtered := make([]*models.ResultEntry, 0, len(items))
	for _, item := range items {
		if item == nil || item.State != state {
			continue
		}
		filtered = append(filtered, item)
	}

	return filtered, nil
}

//nolint:whitespace // multiline signature style
func (r *resultEntriesEntityRepo) DeleteByRaceID(
	ctx context.Context,
	raceID int32,
) error {
	items, err := r.LoadAll(ctx)
	if err != nil {
		return err
	}

	for _, item := range items {
		if item == nil || item.RaceID != raceID {
			continue
		}
		if err := r.DeleteByID(ctx, item.ID); err != nil {
			return err
		}
	}

	return nil
}

type bookingEntriesEntityRepo struct {
	*mapEntityRepo[models.BookingEntry, models.BookingEntrySetter]
}

//nolint:whitespace // multiline signature style
func (r *bookingEntriesEntityRepo) LoadByEventID(
	ctx context.Context,
	eventID int32,
) ([]*models.BookingEntry, error) {
	items, err := r.LoadAll(ctx)
	if err != nil {
		return nil, err
	}

	filtered := make([]*models.BookingEntry, 0, len(items))
	for _, item := range items {
		if item == nil || item.EventID != eventID {
			continue
		}
		filtered = append(filtered, item)
	}

	return filtered, nil
}

//nolint:whitespace // multiline signature style
func (r *bookingEntriesEntityRepo) DeleteByEventIDAndTargetType(
	_ context.Context,
	eventID int32,
	targetType string,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for id, item := range r.data {
		if item.EventID == eventID && string(item.TargetType) == targetType {
			delete(r.data, id)
		}
	}

	return nil
}

//nolint:whitespace // multiline signature style
func (r *bookingEntriesEntityRepo) DeleteByEventIDAndSourceType(
	_ context.Context,
	eventID int32,
	sourceType string,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for id, item := range r.data {
		if item.EventID == eventID && string(item.SourceType) == sourceType {
			delete(r.data, id)
		}
	}

	return nil
}

type eventProcessingAuditEntityRepo struct {
	*mapEntityRepo[models.EventProcessingAudit, models.EventProcessingAuditSetter]
}

//nolint:whitespace // multiline signature style
func (r *eventProcessingAuditEntityRepo) LoadByEventID(
	ctx context.Context,
	eventID int32,
) ([]*models.EventProcessingAudit, error) {
	items, err := r.LoadAll(ctx)
	if err != nil {
		return nil, err
	}

	filtered := make([]*models.EventProcessingAudit, 0, len(items))
	for _, item := range items {
		if item == nil || item.EventID != eventID {
			continue
		}
		filtered = append(filtered, item)
	}

	return filtered, nil
}

//nolint:whitespace // multiline signature style
func (r *eventProcessingAuditEntityRepo) DeleteByImportBatchID(
	ctx context.Context,
	importBatchID int32,
) error {
	items, err := r.LoadAll(ctx)
	if err != nil {
		return err
	}

	for _, item := range items {
		if item == nil || item.ImportBatchID.IsNull() ||
			item.ImportBatchID.MustGet() != importBatchID {
			continue
		}
		if err := r.DeleteByID(ctx, item.ID); err != nil {
			return err
		}
	}

	return nil
}

type seasonDriverStandingsEntityRepo struct {
	*mapEntityRepo[models.SeasonDriverStanding, models.SeasonDriverStandingSetter]
}
type seasonTeamStandingsEntityRepo struct {
	*mapEntityRepo[models.SeasonTeamStanding, models.SeasonTeamStandingSetter]
}
type eventDriverStandingsEntityRepo struct {
	*mapEntityRepo[models.EventDriverStanding, models.EventDriverStandingSetter]
}
type eventTeamStandingsEntityRepo struct {
	*mapEntityRepo[models.EventTeamStanding, models.EventTeamStandingSetter]
}

type pointSystemsGroup struct {
	pointSystems pointsystems.PointSystemsRepository
	pointRules   pointsystems.PointRulesRepository
}

func (g *pointSystemsGroup) PointSystems() pointsystems.PointSystemsRepository { return g.pointSystems }

func (g *pointSystemsGroup) PointRules() pointsystems.PointRulesRepository { return g.pointRules }

type driversGroup struct {
	drivers                 drivers.DriversRepository
	simulationDriverAliases drivers.SimulationDriverAliasesRepository
}

func (g *driversGroup) Drivers() drivers.DriversRepository { return g.drivers }
func (g *driversGroup) SimulationDriverAliases() drivers.SimulationDriverAliasesRepository {
	return g.simulationDriverAliases
}

type tracksGroup struct {
	tracks                       tracks.TracksRepository
	trackLayouts                 tracks.TrackLayoutsRepository
	simulationTrackLayoutAliases tracks.SimulationTrackLayoutAliasesRepository
}

func (g *tracksGroup) Tracks() tracks.TracksRepository             { return g.tracks }
func (g *tracksGroup) TrackLayouts() tracks.TrackLayoutsRepository { return g.trackLayouts }
func (g *tracksGroup) SimulationTrackLayoutAliases() tracks.SimulationTrackLayoutAliasesRepository {
	return g.simulationTrackLayoutAliases
}

type carsGroup struct {
	carManufacturers     cars.CarManufacturersRepository
	carBrands            cars.CarBrandsRepository
	carModels            cars.CarModelsRepository
	simulationCarAliases cars.SimulationCarAliasesRepository
}

func (g *carsGroup) CarManufacturers() cars.CarManufacturersRepository { return g.carManufacturers }
func (g *carsGroup) CarBrands() cars.CarBrandsRepository               { return g.carBrands }
func (g *carsGroup) CarModels() cars.CarModelsRepository               { return g.carModels }
func (g *carsGroup) SimulationCarAliases() cars.SimulationCarAliasesRepository {
	return g.simulationCarAliases
}

type teamsGroup struct {
	teams       teams.TeamsRepository
	teamDrivers teams.TeamDriversRepository
}

func (g *teamsGroup) Teams() teams.TeamsRepository             { return g.teams }
func (g *teamsGroup) TeamDrivers() teams.TeamDriversRepository { return g.teamDrivers }

type racesGroup struct {
	races     races.RacesRepository
	raceGrids races.RaceGridsRepository
}

func (g *racesGroup) Races() races.RacesRepository         { return g.races }
func (g *racesGroup) RaceGrids() races.RaceGridsRepository { return g.raceGrids }

type standingsGroup struct {
	seasonDriver standings.SeasonDriverStandingsRepository
	seasonTeam   standings.SeasonTeamStandingsRepository
	eventDriver  standings.EventDriverStandingsRepository
	eventTeam    standings.EventTeamStandingsRepository
}

func (g *standingsGroup) SeasonDriverStandings() standings.SeasonDriverStandingsRepository {
	return g.seasonDriver
}

func (g *standingsGroup) SeasonTeamStandings() standings.SeasonTeamStandingsRepository {
	return g.seasonTeam
}

func (g *standingsGroup) EventDriverStandings() standings.EventDriverStandingsRepository {
	return g.eventDriver
}

func (g *standingsGroup) EventTeamStandings() standings.EventTeamStandingsRepository {
	return g.eventTeam
}

type repository struct {
	racingSims           racingsims.Repository
	pointSystems         pointsystems.Repository
	drivers              drivers.Repository
	tracks               tracks.Repository
	cars                 cars.Repository
	series               series.Repository
	seasons              seasons.Repository
	events               events.Repository
	races                races.Repository
	teams                teams.Repository
	importBatches        importbatches.Repository
	resultEntries        resultentries.Repository
	bookingEntries       bookingentries.Repository
	eventProcessingAudit eventprocessingaudit.Repository
	standings            standings.Repository
}

// New returns an in-memory repository pre-populated with consistent sample data.
func New() rootrepo.Repository {
	baseTime := time.Date(2026, time.March, 1, 12, 0, 0, 0, time.UTC)
	creator := "testsupport"

	racingSimRepo := &racingSimsEntityRepo{
		newMapEntityRepo(
			func(m *models.RacingSim) int32 { return m.ID },
			func(m *models.RacingSim, id int32) { m.ID = id },
			func(m *models.RacingSim, s *models.RacingSimSetter) { s.Overwrite(m) },
			&models.RacingSim{
				ID:                     1,
				FrontendID:             mustUUID("00000000-0000-0000-0000-000000000001"),
				Name:                   "iRacing",
				SupportedImportFormats: pq.StringArray{"csv"},
				IsActive:               true,
				CreatedAt:              baseTime,
				UpdatedAt:              baseTime,
				CreatedBy:              creator,
				UpdatedBy:              creator,
			},
		),
	}
	pointSystemRepo := &pointSystemsEntityRepo{
		newMapEntityRepo(
			func(m *models.PointSystem) int32 { return m.ID },
			func(m *models.PointSystem, id int32) { m.ID = id },
			func(m *models.PointSystem, s *models.PointSystemSetter) { s.Overwrite(m) },
			&models.PointSystem{
				ID:         1,
				FrontendID: mustUUID("00000000-0000-0000-0000-000000000002"),
				Name:       "Default Points",
				IsActive:   true,
				CreatedAt:  baseTime,
				UpdatedAt:  baseTime,
				CreatedBy:  creator,
				UpdatedBy:  creator,
			},
		),
	}
	pointRuleRepo := &pointRulesEntityRepo{
		newMapEntityRepo(
			func(m *models.PointRule) int32 { return m.ID },
			func(m *models.PointRule, id int32) { m.ID = id },
			func(m *models.PointRule, s *models.PointRuleSetter) { s.Overwrite(m) },
			&models.PointRule{
				ID:            1,
				PointSystemID: 1,
				CreatedAt:     baseTime,
				UpdatedAt:     baseTime,
				CreatedBy:     creator,
				UpdatedBy:     creator,
			},
		),
	}
	driverRepo := &driversEntityRepo{
		newMapEntityRepo(
			func(m *models.Driver) int32 { return m.ID },
			func(m *models.Driver, id int32) { m.ID = id },
			func(m *models.Driver, s *models.DriverSetter) { s.Overwrite(m) },
			&models.Driver{
				ID:         1,
				FrontendID: mustUUID("00000000-0000-0000-0000-000000000003"),
				ExternalID: "drv-001",
				Name:       "Alex Driver",
				IsActive:   true,
				CreatedAt:  baseTime,
				UpdatedAt:  baseTime,
				CreatedBy:  creator,
				UpdatedBy:  creator,
			},
		),
	}
	simulationDriverAliasRepo := &simulationDriverAliasesEntityRepo{
		newMapEntityRepo(
			func(m *models.SimulationDriverAlias) int32 { return m.ID },
			func(m *models.SimulationDriverAlias, id int32) { m.ID = id },
			func(m *models.SimulationDriverAlias, s *models.SimulationDriverAliasSetter) { s.Overwrite(m) },
			&models.SimulationDriverAlias{
				ID:                 1,
				DriverID:           1,
				SimulationID:       1,
				SimulationDriverID: "alex-ir-01",
				CreatedAt:          baseTime,
				UpdatedAt:          baseTime,
				CreatedBy:          creator,
				UpdatedBy:          creator,
			},
		),
	}
	trackRepo := &tracksEntityRepo{
		newMapEntityRepo(
			func(m *models.Track) int32 { return m.ID },
			func(m *models.Track, id int32) { m.ID = id },
			func(m *models.Track, s *models.TrackSetter) { s.Overwrite(m) },
			&models.Track{
				ID:        1,
				Name:      "Spa-Francorchamps",
				IsActive:  true,
				CreatedAt: baseTime,
				UpdatedAt: baseTime,
				CreatedBy: creator,
				UpdatedBy: creator,
			},
		),
	}
	trackLayoutRepo := &trackLayoutsEntityRepo{
		newMapEntityRepo(
			func(m *models.TrackLayout) int32 { return m.ID },
			func(m *models.TrackLayout, id int32) { m.ID = id },
			func(m *models.TrackLayout, s *models.TrackLayoutSetter) { s.Overwrite(m) },
			&models.TrackLayout{
				ID:        1,
				TrackID:   1,
				Name:      "Grand Prix",
				IsActive:  true,
				CreatedAt: baseTime,
				UpdatedAt: baseTime,
				CreatedBy: creator,
				UpdatedBy: creator,
			},
		),
	}
	simulationTrackLayoutAliasRepo := &simulationTrackLayoutAliasesEntityRepo{
		newMapEntityRepo(
			func(m *models.SimulationTrackLayoutAlias) int32 { return m.ID },
			func(m *models.SimulationTrackLayoutAlias, id int32) { m.ID = id },
			func(m *models.SimulationTrackLayoutAlias, s *models.SimulationTrackLayoutAliasSetter) { s.Overwrite(m) },
			&models.SimulationTrackLayoutAlias{
				ID:            1,
				TrackLayoutID: 1,
				SimulationID:  1,
				ExternalName:  "spa gp",
				CreatedAt:     baseTime,
				UpdatedAt:     baseTime,
				CreatedBy:     creator,
				UpdatedBy:     creator,
			},
		),
	}
	carManufacturerRepo := &carManufacturersEntityRepo{
		newMapEntityRepo(
			func(m *models.CarManufacturer) int32 { return m.ID },
			func(m *models.CarManufacturer, id int32) { m.ID = id },
			func(m *models.CarManufacturer, s *models.CarManufacturerSetter) { s.Overwrite(m) },
			&models.CarManufacturer{
				ID:        1,
				Name:      "Porsche",
				IsActive:  true,
				CreatedAt: baseTime,
				UpdatedAt: baseTime,
				CreatedBy: creator,
				UpdatedBy: creator,
			},
		),
	}
	carBrandRepo := &carBrandsEntityRepo{
		newMapEntityRepo(
			func(m *models.CarBrand) int32 { return m.ID },
			func(m *models.CarBrand, id int32) { m.ID = id },
			func(m *models.CarBrand, s *models.CarBrandSetter) { s.Overwrite(m) },
			&models.CarBrand{
				ID:             1,
				ManufacturerID: 1,
				Name:           "Porsche Motorsport",
				IsActive:       true,
				CreatedAt:      baseTime,
				UpdatedAt:      baseTime,
				CreatedBy:      creator,
				UpdatedBy:      creator,
			},
		),
	}
	carModelRepo := &carModelsEntityRepo{
		mapEntityRepo: newMapEntityRepo(
			func(m *models.CarModel) int32 { return m.ID },
			func(m *models.CarModel, id int32) { m.ID = id },
			func(m *models.CarModel, s *models.CarModelSetter) { s.Overwrite(m) },
			&models.CarModel{
				ID:        1,
				BrandID:   1,
				Name:      "911 GT3 R",
				IsActive:  true,
				CreatedAt: baseTime,
				UpdatedAt: baseTime,
				CreatedBy: creator,
				UpdatedBy: creator,
			},
		),
		brands: carBrandRepo,
	}
	simulationCarAliasRepo := &simulationCarAliasesEntityRepo{
		newMapEntityRepo(
			func(m *models.SimulationCarAlias) int32 { return m.ID },
			func(m *models.SimulationCarAlias, id int32) { m.ID = id },
			func(m *models.SimulationCarAlias, s *models.SimulationCarAliasSetter) { s.Overwrite(m) },
			&models.SimulationCarAlias{
				ID:           1,
				CarModelID:   1,
				SimulationID: 1,
				ExternalName: "Porsche 911 GT3 R",
				CreatedAt:    baseTime,
				UpdatedAt:    baseTime,
				CreatedBy:    creator,
				UpdatedBy:    creator,
			},
		),
	}
	seriesRepo := &seriesEntityRepo{
		newMapEntityRepo(
			func(m *models.Series) int32 { return m.ID },
			func(m *models.Series, id int32) { m.ID = id },
			func(m *models.Series, s *models.SeriesSetter) { s.Overwrite(m) },
			&models.Series{
				ID:           1,
				FrontendID:   mustUUID("00000000-0000-0000-0000-000000000004"),
				SimulationID: 1,
				Name:         "GT Sprint",
				IsActive:     true,
				CreatedAt:    baseTime,
				UpdatedAt:    baseTime,
				CreatedBy:    creator,
				UpdatedBy:    creator,
			},
		),
	}
	seasonsRepo := &seasonsEntityRepo{
		newMapEntityRepo(
			func(m *models.Season) int32 { return m.ID },
			func(m *models.Season, id int32) { m.ID = id },
			func(m *models.Season, s *models.SeasonSetter) { s.Overwrite(m) },
			&models.Season{
				ID:            1,
				FrontendID:    mustUUID("00000000-0000-0000-0000-000000000005"),
				SeriesID:      1,
				PointSystemID: 1,
				Name:          "2026 Spring",
				HasTeams:      true,
				SkipEvents:    1,
				Status:        "active",
				CreatedAt:     baseTime,
				UpdatedAt:     baseTime,
				CreatedBy:     creator,
				UpdatedBy:     creator,
			},
		),
	}
	eventsRepo := &eventsEntityRepo{
		newMapEntityRepo(
			func(m *models.Event) int32 { return m.ID },
			func(m *models.Event, id int32) { m.ID = id },
			func(m *models.Event, s *models.EventSetter) { s.Overwrite(m) },
			&models.Event{
				ID:              1,
				FrontendID:      mustUUID("00000000-0000-0000-0000-000000000006"),
				SeasonID:        1,
				TrackLayoutID:   1,
				Name:            "Round 1",
				EventDate:       baseTime,
				Status:          "scheduled",
				ProcessingState: "pending",
				CreatedAt:       baseTime,
				UpdatedAt:       baseTime,
				CreatedBy:       creator,
				UpdatedBy:       creator,
			},
		),
	}
	racesRepo := &racesEntityRepo{
		newMapEntityRepo(
			func(m *models.Race) int32 { return m.ID },
			func(m *models.Race, id int32) { m.ID = id },
			func(m *models.Race, s *models.RaceSetter) { s.Overwrite(m) },
			&models.Race{
				ID:          1,
				EventID:     1,
				Name:        "Feature Race",
				SessionType: "race",
				SequenceNo:  1,
				CreatedAt:   baseTime,
				UpdatedAt:   baseTime,
				CreatedBy:   creator,
				UpdatedBy:   creator,
			},
		),
	}
	raceGridsRepo := &raceGridsEntityRepo{
		newMapEntityRepo(
			func(m *models.RaceGrid) int32 { return m.ID },
			func(m *models.RaceGrid, id int32) { m.ID = id },
			func(m *models.RaceGrid, s *models.RaceGridSetter) { s.Overwrite(m) },
		),
	}
	teamsRepo := &teamsEntityRepo{
		newMapEntityRepo(
			func(m *models.Team) int32 { return m.ID },
			func(m *models.Team, id int32) { m.ID = id },
			func(m *models.Team, s *models.TeamSetter) { s.Overwrite(m) },
			&models.Team{
				ID:         1,
				FrontendID: mustUUID("00000000-0000-0000-0000-000000000007"),
				SeasonID:   1,
				Name:       "Velocity Racing",
				IsActive:   true,
				CreatedAt:  baseTime,
				UpdatedAt:  baseTime,
				CreatedBy:  creator,
				UpdatedBy:  creator,
			},
		),
	}
	teamDriversRepo := &teamDriversEntityRepo{
		newMapEntityRepo(
			func(m *models.TeamDriver) int32 { return m.ID },
			func(m *models.TeamDriver, id int32) { m.ID = id },
			func(m *models.TeamDriver, s *models.TeamDriverSetter) { s.Overwrite(m) },
			&models.TeamDriver{
				ID:         1,
				FrontendID: mustUUID("00000000-0000-0000-0000-000000000008"),
				TeamID:     1,
				DriverID:   1,
				JoinedAt:   baseTime,
				CreatedAt:  baseTime,
				UpdatedAt:  baseTime,
				CreatedBy:  creator,
				UpdatedBy:  creator,
			},
		),
	}
	importBatchesRepo := &importBatchesEntityRepo{
		newMapEntityRepo(
			func(m *models.ImportBatch) int32 { return m.ID },
			func(m *models.ImportBatch, id int32) { m.ID = id },
			func(m *models.ImportBatch, s *models.ImportBatchSetter) { s.Overwrite(m) },
			&models.ImportBatch{
				ID:              1,
				FrontendID:      mustUUID("00000000-0000-0000-0000-000000000009"),
				RaceID:          1,
				ImportFormat:    mytypes.ImportFormat("csv"),
				Payload:         []byte("sample import payload"),
				ProcessingState: "queued",
				CreatedAt:       baseTime,
				UpdatedAt:       baseTime,
				CreatedBy:       creator,
				UpdatedBy:       creator,
			},
		),
	}
	resultEntriesRepo := &resultEntriesEntityRepo{
		newMapEntityRepo(
			func(m *models.ResultEntry) int32 { return m.ID },
			func(m *models.ResultEntry, id int32) { m.ID = id },
			func(m *models.ResultEntry, s *models.ResultEntrySetter) { s.Overwrite(m) },
			&models.ResultEntry{
				ID:             1,
				FrontendID:     mustUUID("00000000-0000-0000-0000-000000000010"),
				RaceID:         1,
				RawDriverName:  null.From("Alex Driver"),
				FinishPosition: 1,
				LapsCompleted:  25,
				State:          "pending",
				CreatedAt:      baseTime,
				UpdatedAt:      baseTime,
				CreatedBy:      creator,
				UpdatedBy:      creator,
			},
		),
	}
	bookingEntriesRepo := &bookingEntriesEntityRepo{
		newMapEntityRepo(
			func(m *models.BookingEntry) int32 { return m.ID },
			func(m *models.BookingEntry, id int32) { m.ID = id },
			func(m *models.BookingEntry, s *models.BookingEntrySetter) { s.Overwrite(m) },
			&models.BookingEntry{
				ID:          1,
				FrontendID:  mustUUID("00000000-0000-0000-0000-000000000011"),
				EventID:     1,
				TargetType:  mytypes.TargetType("driver"),
				SourceType:  mytypes.SourceType("result"),
				Points:      25,
				Description: "Race points",
				IsManual:    false,
				CreatedAt:   baseTime,
				UpdatedAt:   baseTime,
				CreatedBy:   creator,
				UpdatedBy:   creator,
			},
		),
	}
	auditRepo := &eventProcessingAuditEntityRepo{
		newMapEntityRepo(
			func(m *models.EventProcessingAudit) int32 { return m.ID },
			func(m *models.EventProcessingAudit, id int32) { m.ID = id },
			func(m *models.EventProcessingAudit, s *models.EventProcessingAuditSetter) { s.Overwrite(m) },
			&models.EventProcessingAudit{
				ID:         1,
				FrontendID: mustUUID("00000000-0000-0000-0000-000000000012"),
				EventID:    1,
				ToState:    "queued",
				Action:     "created",
				CreatedAt:  baseTime,
				UpdatedAt:  baseTime,
				CreatedBy:  creator,
				UpdatedBy:  creator,
			},
		),
	}
	seasonDriverStandingsRepo := &seasonDriverStandingsEntityRepo{
		newMapEntityRepo(
			func(m *models.SeasonDriverStanding) int32 { return m.ID },
			func(m *models.SeasonDriverStanding, id int32) { m.ID = id },
			func(m *models.SeasonDriverStanding, s *models.SeasonDriverStandingSetter) { s.Overwrite(m) },
			&models.SeasonDriverStanding{
				ID:              1,
				FrontendID:      mustUUID("00000000-0000-0000-0000-000000000013"),
				SeasonID:        1,
				DriverID:        1,
				Position:        1,
				TotalPoints:     25,
				DroppedEventIds: pq.Int32Array{},
				LastRebuiltAt:   baseTime,
				CreatedAt:       baseTime,
				UpdatedAt:       baseTime,
				CreatedBy:       creator,
				UpdatedBy:       creator,
			},
		),
	}
	seasonTeamStandingsRepo := &seasonTeamStandingsEntityRepo{
		newMapEntityRepo(
			func(m *models.SeasonTeamStanding) int32 { return m.ID },
			func(m *models.SeasonTeamStanding, id int32) { m.ID = id },
			func(m *models.SeasonTeamStanding, s *models.SeasonTeamStandingSetter) { s.Overwrite(m) },
			&models.SeasonTeamStanding{
				ID:              1,
				FrontendID:      mustUUID("00000000-0000-0000-0000-000000000014"),
				SeasonID:        1,
				TeamID:          1,
				Position:        1,
				TotalPoints:     25,
				DroppedEventIds: pq.Int32Array{},
				LastRebuiltAt:   baseTime,
				CreatedAt:       baseTime,
				UpdatedAt:       baseTime,
				CreatedBy:       creator,
				UpdatedBy:       creator,
			},
		),
	}
	eventDriverStandingsRepo := &eventDriverStandingsEntityRepo{
		newMapEntityRepo(
			func(m *models.EventDriverStanding) int32 { return m.ID },
			func(m *models.EventDriverStanding, id int32) { m.ID = id },
			func(m *models.EventDriverStanding, s *models.EventDriverStandingSetter) { s.Overwrite(m) },
			&models.EventDriverStanding{
				ID:              1,
				FrontendID:      mustUUID("00000000-0000-0000-0000-000000000015"),
				EventID:         1,
				SeasonID:        1,
				DriverID:        1,
				Position:        1,
				TotalPoints:     25,
				DroppedEventIds: pq.Int32Array{},
				CreatedAt:       baseTime,
				UpdatedAt:       baseTime,
				CreatedBy:       creator,
				UpdatedBy:       creator,
			},
		),
	}
	eventTeamStandingsRepo := &eventTeamStandingsEntityRepo{
		newMapEntityRepo(
			func(m *models.EventTeamStanding) int32 { return m.ID },
			func(m *models.EventTeamStanding, id int32) { m.ID = id },
			func(m *models.EventTeamStanding, s *models.EventTeamStandingSetter) { s.Overwrite(m) },
			&models.EventTeamStanding{
				ID:              1,
				FrontendID:      mustUUID("00000000-0000-0000-0000-000000000016"),
				EventID:         1,
				SeasonID:        1,
				TeamID:          1,
				Position:        1,
				TotalPoints:     25,
				DroppedEventIds: pq.Int32Array{},
				CreatedAt:       baseTime,
				UpdatedAt:       baseTime,
				CreatedBy:       creator,
				UpdatedBy:       creator,
			},
		),
	}

	return &repository{
		racingSims: racingSimRepo,
		pointSystems: &pointSystemsGroup{
			pointSystems: pointSystemRepo,
			pointRules:   pointRuleRepo,
		},
		drivers: &driversGroup{
			drivers:                 driverRepo,
			simulationDriverAliases: simulationDriverAliasRepo,
		},
		tracks: &tracksGroup{
			tracks:                       trackRepo,
			trackLayouts:                 trackLayoutRepo,
			simulationTrackLayoutAliases: simulationTrackLayoutAliasRepo,
		},
		cars: &carsGroup{
			carManufacturers:     carManufacturerRepo,
			carBrands:            carBrandRepo,
			carModels:            carModelRepo,
			simulationCarAliases: simulationCarAliasRepo,
		},
		series:               seriesRepo,
		seasons:              seasonsRepo,
		events:               eventsRepo,
		races:                &racesGroup{races: racesRepo, raceGrids: raceGridsRepo},
		teams:                &teamsGroup{teams: teamsRepo, teamDrivers: teamDriversRepo},
		importBatches:        importBatchesRepo,
		resultEntries:        resultEntriesRepo,
		bookingEntries:       bookingEntriesRepo,
		eventProcessingAudit: auditRepo,
		standings: &standingsGroup{
			seasonDriver: seasonDriverStandingsRepo,
			seasonTeam:   seasonTeamStandingsRepo,
			eventDriver:  eventDriverStandingsRepo,
			eventTeam:    eventTeamStandingsRepo,
		},
	}
}

func (r *repository) RacingSims() racingsims.Repository         { return r.racingSims }
func (r *repository) PointSystems() pointsystems.Repository     { return r.pointSystems }
func (r *repository) Drivers() drivers.Repository               { return r.drivers }
func (r *repository) Tracks() tracks.Repository                 { return r.tracks }
func (r *repository) Cars() cars.Repository                     { return r.cars }
func (r *repository) Series() series.Repository                 { return r.series }
func (r *repository) Seasons() seasons.Repository               { return r.seasons }
func (r *repository) Events() events.Repository                 { return r.events }
func (r *repository) Races() races.Repository                   { return r.races }
func (r *repository) Teams() teams.Repository                   { return r.teams }
func (r *repository) ImportBatches() importbatches.Repository   { return r.importBatches }
func (r *repository) ResultEntries() resultentries.Repository   { return r.resultEntries }
func (r *repository) BookingEntries() bookingentries.Repository { return r.bookingEntries }
func (r *repository) EventProcessingAudit() eventprocessingaudit.Repository {
	return r.eventProcessingAudit
}
func (r *repository) Standings() standings.Repository { return r.standings }

func mustUUID(raw string) uuid.UUID {
	parsed, err := uuid.FromString(raw)
	if err != nil {
		panic(err)
	}
	return parsed
}
