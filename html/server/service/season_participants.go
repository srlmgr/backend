package service

import (
	"context"

	"github.com/samber/lo"

	dbModels "github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/html/server/model"
)

//nolint:whitespace,funlen //editor/linter issue
func (s *serviceImpl) GetSeasonParticipants(
	ctx context.Context,
	seasonID int) (
	*model.SeasonParticipantsContainer, error,
) {
	season, _ := s.r.Seasons().LoadByID(ctx, int32(seasonID))

	seasonDrivers, err := s.svc.ParticipantsService().GetDrivers(ctx, seasonID)
	if err != nil {
		return nil, err
	}
	teams, err := s.svc.ParticipantsService().GetTeams(ctx, seasonID)
	if err != nil {
		return nil, err
	}

	seasonContainer, err := s.GetSeasonList(ctx, int(season.SeriesID))
	if err != nil {
		return nil, err
	}
	ret := &model.SeasonParticipantsContainer{
		SeasonsContainer: seasonContainer,
		Season:           season,
	}

	ec, err := newSeasonEntryComposer(s.r, ctx, season.ID)
	if err != nil {
		return nil, err
	}
	if season.IsTeamBased {
		teamLookup := ec.CreateTeamLookup(ctx)
		ret.PrimaryParticipants = lo.Map(teams,
			func(d *dbModels.Team, _ int) model.Participant {
				return entryBackedParticipant{
					entry: teamLookup[d.ID],
				}
			})
	} else {
		driverLookup := ec.CreateDriverLookup(ctx)
		ret.PrimaryParticipants = lo.Map(seasonDrivers,
			func(d *dbModels.SeasonDriver, _ int) model.Participant {
				return entryBackedParticipant{
					entry: driverLookup[d.DriverID],
				}
			})
	}
	return ret, nil
}

type entryBackedParticipant struct {
	entry *model.Entry
}

func (p entryBackedParticipant) ID() int32 {
	return p.entry.ID
}

func (p entryBackedParticipant) CarNumber() string {
	return p.entry.CarNum
}

func (p entryBackedParticipant) CarClass() string {
	return p.entry.CarClass
}

func (p entryBackedParticipant) CarName() string {
	return p.entry.CarName
}

func (p entryBackedParticipant) Name() string {
	return p.entry.Name
}

var _ model.Participant = (*entryBackedParticipant)(nil)
