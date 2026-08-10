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

	carModelVariants, err := s.r.Cars().CarModelVariants().LoadBySeasonID(ctx, season.ID)
	if err != nil {
		return nil, err
	}
	cmvLookup := make(map[int32]*dbModels.CarModelVariant)
	for _, cmv := range carModelVariants {
		cmvLookup[cmv.ID] = cmv
	}
	ret := &model.SeasonParticipantsContainer{
		SeasonsContainer: seasonContainer,
		Season:           season,
	}
	seasonCarClasses, err := s.r.Cars().CarClasses().LoadBySeasonID(ctx, season.ID)
	if err != nil {
		return nil, err
	}
	ccLookup := make(map[int32]*dbModels.CarClass)
	cmv2ccLookup := make(map[int32]*dbModels.CarClass)
	for _, cc := range seasonCarClasses {
		ccLookup[cc.ID] = cc
		cmvInClass, err := s.r.Cars().CarModelVariants().LoadByCarClassID(ctx, cc.ID)
		if err != nil {
			return nil, err
		}
		for _, cmv := range cmvInClass {
			cmv2ccLookup[cmv.ID] = cc
		}
	}
	if season.IsTeamBased {
		ret.PrimaryParticipants = lo.Map(teams,
			func(d *dbModels.Team, _ int) model.Participant {
				cmv, _ := lo.Coalesce(
					cmvLookup[d.CarModelVariantID.GetOrZero()],
					&dbModels.CarModelVariant{Name: "n.a."})

				cClass, _ := lo.Coalesce(
					cmv2ccLookup[cmv.ID],
					&dbModels.CarClass{Name: "n.a."})
				return &participantImpl{
					id:        d.ID,
					carNumber: d.CarNumber.GetOr("n.a."),
					name:      d.Name,
					carClass:  cClass.Name,
					carName:   cmv.Name,
				}
			})
	} else {
		ids := lo.Map(seasonDrivers, func(d *dbModels.SeasonDriver, _ int) int32 {
			return d.DriverID
		})
		driverMap, err := s.resolveForDriverIDs(ctx, ids)
		if err != nil {
			return nil, err
		}
		ret.PrimaryParticipants = lo.Map(seasonDrivers,
			func(d *dbModels.SeasonDriver, _ int) model.Participant {
				cmv, _ := lo.Coalesce(
					cmvLookup[d.CarModelVariantID],
					&dbModels.CarModelVariant{Name: "n.a."})
				cClass, _ := lo.Coalesce(
					cmv2ccLookup[cmv.ID],
					&dbModels.CarClass{Name: "n.a."})
				return &participantImpl{
					id:        d.DriverID,
					carNumber: d.CarNumber,
					carName:   cmvLookup[d.CarModelVariantID].Name,
					name:      driverMap[d.DriverID],
					carClass:  cClass.Name,
				}
			})
	}
	return ret, nil
}

type participantImpl struct {
	id        int32
	carNumber string
	name      string
	carClass  string
	carName   string
}

func (p *participantImpl) ID() int32 {
	return p.id
}

func (p *participantImpl) CarNumber() string {
	return p.carNumber
}

func (p *participantImpl) CarClass() string {
	return p.carClass
}

func (p *participantImpl) CarName() string {
	return p.carName
}

func (p *participantImpl) Name() string {
	return p.name
}

var _ model.Participant = (*participantImpl)(nil)
