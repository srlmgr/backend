package importer

import (
	"context"
	"errors"
	"fmt"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/repository"
	"github.com/srlmgr/backend/services/importsvc/processor"
)

// RepositoryEntityResolver resolves simulation-specific identifiers via repositories.
type RepositoryEntityResolver struct {
	ctx   context.Context
	repos repository.Repository
	sim   *models.RacingSim
	epi   *processor.EventProcInfo
}

// NewRepositoryEntityResolver returns a repository-backed EntityResolver.
//
//nolint:whitespace // editor/linter issue
func NewRepositoryEntityResolver(
	ctx context.Context,
	repos repository.Repository,
	epi *processor.EventProcInfo,
	sim *models.RacingSim,
) *RepositoryEntityResolver {
	return &RepositoryEntityResolver{
		ctx:   ctx,
		repos: repos,
		sim:   sim,
		epi:   epi,
	}
}

//nolint:whitespace // editor/linter issue
func (r *RepositoryEntityResolver) ResolveTrack(
	simTrackID, simTrackName string,
) (uint32, error) {
	if r.sim == nil {
		return 0, fmt.Errorf("resolve track: simulation is not set")
	}

	resolveArg := func(arg string) (uint32, error) {
		alias, err := r.repos.Tracks().SimulationTrackLayoutAliases().FindBySimID(
			r.ctx,
			r.sim.ID,
			simTrackID, simTrackName,
		)
		if err != nil {
			return 0, err
		}

		return uint32(alias.TrackLayoutID), nil
	}

	return r.resolveBy("track", simTrackID, simTrackName, resolveArg)
}

//nolint:whitespace // editor/linter issue
func (r *RepositoryEntityResolver) ResolveDriver(
	simDriverID, simDriverName string,
) (uint32, error) {
	if r.sim == nil {
		return 0, fmt.Errorf("resolve driver: simulation is not set")
	}

	resolveArg := func(arg string) (uint32, error) {
		driverIDBySim, err := r.repos.Drivers().SimulationDriverAliases().FindBySimID(
			r.ctx,
			r.sim.ID,
			simDriverID, simDriverName,
		)
		if err != nil {
			return 0, err
		}

		return uint32(driverIDBySim.DriverID), nil
	}

	driverID, err := r.repos.Drivers().Drivers().FindByName(
		r.ctx, simDriverName)
	if err == nil {
		return uint32(driverID.ID), nil
	}

	return r.resolveBy("driver", simDriverID, simDriverName, resolveArg)
}

//nolint:whitespace // editor/linter issue
func (r *RepositoryEntityResolver) ResolveCar(
	simCarID, simCarName string,
) (uint32, error) {
	if r.sim == nil {
		return 0, fmt.Errorf("resolve car: simulation is not set")
	}

	resolveArg := func(arg string) (uint32, error) {
		alias, err := r.repos.Cars().SimulationCarAliases().FindBySimID(
			r.ctx,
			r.sim.ID,
			simCarID, simCarName,
		)
		if err != nil {
			return 0, err
		}

		return uint32(alias.CarModelID), nil
	}

	return r.resolveBy("car", simCarID, simCarName, resolveArg)
}

//nolint:whitespace // editor/linter issue
func (r *RepositoryEntityResolver) ResolveCarClass(
	ownCarID uint32,
) (uint32, error) {
	if r.epi.Season == nil {
		return 0, fmt.Errorf("resolve car class: season is not set")
	}

	carClass, err := r.repos.Queries().QueryCarClasses().FindBySeasonAndCarModel(
		r.ctx,
		r.epi.Season.ID,
		int32(ownCarID),
	)
	if err != nil {
		return 0, fmt.Errorf("resolve car class: %w", err)
	}
	return uint32(carClass.ID), nil
	// TODO: lookup in classes defined for this season
}

//nolint:whitespace // editor/linter issue
func (r *RepositoryEntityResolver) ResolveTeam(
	ownDriverID uint32,
) (uint32, error) {
	td, err := r.repos.Queries().QueryTeamDrivers().FindBySeasonAndDriver(
		r.ctx, r.epi.Season.ID, int32(ownDriverID))
	if err != nil {
		return 0, fmt.Errorf("resolve team: %w", err)
	}
	return uint32(td.TeamID), nil
}

//nolint:whitespace // editor/linter issue
func (r *RepositoryEntityResolver) resolveBy(
	kind string,
	simID string,
	simName string,
	resolveArg func(arg string) (uint32, error),
) (uint32, error) {
	entityID, err := resolveArg(simID)
	if err == nil {
		return entityID, nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return 0, fmt.Errorf("resolve %s by id %q: %w", kind, simID, err)
	}

	entityID, err = resolveArg(simName)
	if err == nil {
		return entityID, nil
	}
	if errors.Is(err, repository.ErrNotFound) {
		return 0, fmt.Errorf(
			"resolve %s: no %s found for simulation %q by id %q or name %q",
			kind,
			kind,
			r.sim.Name,
			simID,
			simName,
		)
	}

	return 0, fmt.Errorf("resolve %s by name %q: %w", kind, simName, err)
}
