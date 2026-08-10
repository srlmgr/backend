package service

import (
	"context"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/trace"

	dbModels "github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/html/server/model"
	"github.com/srlmgr/backend/log"
	"github.com/srlmgr/backend/repository"
	gs "github.com/srlmgr/backend/service"
	gsImpl "github.com/srlmgr/backend/service/impl"
	svcStandings "github.com/srlmgr/backend/service/standings"
)

type (
	Service interface {
		// Standings
		GetSeasonStandings(
			ctx context.Context,
			seasonID int,
			skipMode svcStandings.SkipModeType,
		) (*model.SeasonStandingsContainer, error)
		GetEventStandings(
			ctx context.Context,
			eventID int,
			skipMode svcStandings.SkipModeType,
		) (*model.SeasonStandingsContainer, error)

		// Get series list
		GetSeriesList(ctx context.Context) (*model.SeriesContainer, error)
		// Get season list for series
		GetSeasonList(ctx context.Context, seriesID int) (*model.SeasonsContainer, error)
		// Get season details
		GetSeason(ctx context.Context, seasonID int) (*dbModels.Season, error)
		// Get car classes for season
		GetSeasonCarClasses(ctx context.Context, seasonID int) ([]*model.CarClass, error)
		// Get events for season
		// Get participants for season
		GetSeasonParticipants(
			ctx context.Context,
			seasonID int,
		) (*model.SeasonParticipantsContainer, error)
		// Get results overview for season by event
		// classID is optional. If classID is 0, then the overview will be for all classes.
		GetResultsOverview(
			ctx context.Context,
			seasonID, classID int,
		) (*model.SeasonResultsOverviewContainer, error)
	}
	serviceImpl struct {
		r           repository.Repository
		svc         gs.Service
		logger      *log.Logger
		externalURL string
	}
)

var _ Service = (*serviceImpl)(nil)

//nolint:whitespace //editor/linter issue
func NewService(
	r repository.Repository,
	logger *log.Logger,
	tracer trace.Tracer,
	externalURL string,
) Service {
	return &serviceImpl{
		r:           r,
		logger:      logger,
		svc:         gsImpl.New(r, logger, tracer),
		externalURL: externalURL,
	}
}

//nolint:whitespace //editor/linter issue
func (s *serviceImpl) GetSeriesList(
	ctx context.Context,
) (
	*model.SeriesContainer, error,
) {
	series, err := s.r.Series().LoadAll(ctx)
	if err != nil {
		return nil, err
	}
	return &model.SeriesContainer{
		Serieses: lo.Map(series, func(s *dbModels.Series, _ int) *model.Series {
			return &model.Series{
				ID:   int(s.ID),
				Name: s.Name,
			}
		}),
	}, nil
}

//nolint:whitespace //editor/linter issue
func (s *serviceImpl) GetSeasonList(
	ctx context.Context,
	seriesID int,
) (
	*model.SeasonsContainer, error,
) {
	seriesContainer, err := s.GetSeriesList(ctx)
	if err != nil {
		return nil, err
	}
	seasons, err := s.r.Seasons().LoadBySeriesID(ctx, int32(seriesID))
	if err != nil {
		return nil, err
	}
	return &model.SeasonsContainer{
		SeriesContainer: seriesContainer,
		Seasons: lo.Map(seasons, func(s *dbModels.Season, _ int) *model.Season {
			return &model.Season{
				ID:   int(s.ID),
				Name: s.Name,
			}
		}),
	}, nil
}

//nolint:whitespace //editor/linter issue
func (s *serviceImpl) GetSeason(
	ctx context.Context,
	seasonID int,
) (
	*dbModels.Season, error,
) {
	return s.r.Seasons().LoadByID(ctx, int32(seasonID))
}

//nolint:whitespace //editor/linter issue
func (s *serviceImpl) GetSeasonCarClasses(
	ctx context.Context,
	seasonID int,
) (
	[]*model.CarClass, error,
) {
	carClasses, err := s.r.Cars().CarClasses().LoadBySeasonID(ctx, int32(seasonID))
	if err != nil {
		return nil, err
	}
	return lo.Map(carClasses, func(cc *dbModels.CarClass, _ int) *model.CarClass {
		return &model.CarClass{
			ID:   int(cc.ID),
			Name: cc.Name,
		}
	}), nil
}

//nolint:whitespace //editor/linter issue
func (s *serviceImpl) GetSeasonStandings(
	ctx context.Context,
	seasonID int,
	skipMode svcStandings.SkipModeType) (
	*model.SeasonStandingsContainer, error,
) {
	standings, err := s.svc.StandingsService().GetSeasonStandings(ctx, seasonID, skipMode)
	if err != nil {
		return nil, err
	}
	return s.prepareStandingsContainer(ctx, standings)
}

//nolint:whitespace //editor/linter issue
func (s *serviceImpl) GetEventStandings(
	ctx context.Context,
	eventID int,
	skipMode svcStandings.SkipModeType) (
	*model.SeasonStandingsContainer, error,
) {
	standings, err := s.svc.StandingsService().GetEventStandings(ctx, eventID, skipMode)
	if err != nil {
		return nil, err
	}
	return s.prepareStandingsContainer(ctx, standings)
}

//nolint:whitespace,funlen //editor/linter issue
func (s *serviceImpl) prepareStandingsContainer(
	ctx context.Context,
	standings *gs.StandingsContainer,
) (*model.SeasonStandingsContainer, error) {
	teamLookup, err := s.resolveForTeam(ctx, int(standings.Season.ID))
	if err != nil {
		return nil, err
	}
	season, _ := s.r.Seasons().LoadByID(ctx, standings.Season.ID)

	seasonContainer, err := s.GetSeasonList(ctx, int(season.SeriesID))
	if err != nil {
		return nil, err
	}
	ret := &model.SeasonStandingsContainer{
		ServiceData:      *standings,
		SeasonsContainer: seasonContainer,
	}
	if standings.Season.IsTeamBased {
		ret.PrimaryLookup = teamLookup
		driverLookup, err := s.resolveTeamDrivers(ctx, int(standings.Season.ID))
		if err != nil {
			return nil, err
		}
		ret.SecondaryLookup = driverLookup
	} else {
		driverLookup, err := s.resolveForDriver(ctx, standings.Primary)
		if err != nil {
			return nil, err
		}
		ret.PrimaryLookup = driverLookup
		ret.SecondaryLookup = teamLookup
	}
	if standings.Season.IsMulticlass {
		carClasses, err := s.r.Cars().CarClasses().LoadBySeasonID(
			ctx, standings.Season.ID)
		if err != nil {
			return nil, err
		}
		ret.CarClasses = lo.Map(carClasses,
			func(cc *dbModels.CarClass, _ int) *model.CarClass {
				return &model.CarClass{
					ID:   int(cc.ID),
					Name: cc.Name,
				}
			})
	}
	if err := s.fillEvents(ctx, ret); err != nil {
		return nil, err
	}
	return ret, nil
}

//nolint:whitespace //editor/linter issue
func (s *serviceImpl) fillEvents(
	ctx context.Context,
	standings *model.SeasonStandingsContainer,
) error {
	events, err := s.r.Events().LoadBySeasonID(ctx, standings.ServiceData.Season.ID)
	if err != nil {
		return err
	}
	standings.Events = lo.Map(events, func(e *dbModels.Event, _ int) *model.Event {
		return &model.Event{
			ID:   int(e.ID),
			Name: e.Name,
			Date: e.EventDate,
		}
	})
	return nil
}

//nolint:whitespace //editor/linter issue
func (s *serviceImpl) resolveForDriver(
	ctx context.Context,
	standings []*gs.Standing) (
	map[int32]string, error,
) {
	ids := lo.Map(standings, func(s *gs.Standing, _ int) int32 {
		return int32(s.ReferenceID)
	})
	return s.resolveForDriverIDs(ctx, ids)
}

//nolint:whitespace //editor/linter issue
func (s *serviceImpl) resolveForDriverIDs(
	ctx context.Context,
	ids []int32) (
	map[int32]string, error,
) {
	drivers, err := s.r.Drivers().Drivers().LoadByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	driverMap := make(map[int32]string)
	for _, d := range drivers {
		driverMap[d.ID] = d.Name
	}
	return driverMap, nil
}

//nolint:whitespace //editor/linter issue
func (s *serviceImpl) resolveForTeam(
	ctx context.Context,
	seasonID int) (
	map[int32]string, error,
) {
	teams, err := s.r.Teams().Teams().LoadBySeasonID(ctx, int32(seasonID))
	if err != nil {
		return nil, err
	}

	teamMap := make(map[int32]string)
	for _, t := range teams {
		teamMap[t.ID] = t.Name
	}
	return teamMap, nil
}

//nolint:whitespace //editor/linter issue
func (s *serviceImpl) resolveTeamDrivers(
	ctx context.Context,
	seasonID int) (
	map[int32]string, error,
) {
	teamDrivers, err := s.r.Queries().QueryTeamDrivers().FindBySeason(ctx, int32(seasonID))
	if err != nil {
		return nil, err
	}
	ids := lo.Map(teamDrivers, func(td *dbModels.TeamDriver, _ int) int32 {
		return td.DriverID
	})
	drivers, err := s.r.Drivers().Drivers().LoadByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	driverMap := make(map[int32]string)
	for _, d := range drivers {
		driverMap[d.ID] = d.Name
	}
	return driverMap, nil
}
