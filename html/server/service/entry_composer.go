package service

import (
	"context"

	"github.com/samber/lo"

	dbModels "github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/html/server/model"
	"github.com/srlmgr/backend/repository"
)

type (
	EntryComposer struct {
		r                repository.Repository
		season           *dbModels.Season
		seasonDrivers    []*dbModels.SeasonDriver
		drivers          []*dbModels.Driver
		teams            []*dbModels.Team
		teamDrivers      []*dbModels.TeamDriver
		events           []*dbModels.Event
		carModelVariants []*dbModels.CarModelVariant
		seasonCarClasses []*dbModels.CarClass
		cmvLookup        map[int32]*dbModels.CarModelVariant
		ccLookup         map[int32]*dbModels.CarClass
		cmv2ccLookup     map[int32]*dbModels.CarClass
	}
)

//nolint:whitespace //editor/linter issue
func newSeasonEntryComposer(
	r repository.Repository,
	ctx context.Context,
	seasonID int32,
) (*EntryComposer, error) {
	ret := &EntryComposer{r: r}
	err := ret.Init(ctx, seasonID)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

//nolint:whitespace,funlen //editor/linter issue
func (c *EntryComposer) Init(
	ctx context.Context, seasonID int32,
) (err error) {
	c.season, err = c.r.Seasons().LoadByID(ctx, seasonID)
	if err != nil {
		return err
	}
	c.seasonDrivers, err = c.r.Drivers().SeasonDrivers().LoadBySeasonID(ctx, seasonID)
	if err != nil {
		return err
	}
	driverIDs := lo.Map(c.seasonDrivers,
		func(item *dbModels.SeasonDriver, _ int) int32 {
			return item.DriverID
		})
	c.drivers, err = c.r.Drivers().Drivers().LoadByIDs(ctx, driverIDs)
	if err != nil {
		return err
	}
	c.teams, err = c.r.Teams().Teams().LoadBySeasonID(ctx, seasonID)
	if err != nil {
		return err
	}
	c.teamDrivers, err = c.r.Queries().QueryTeamDrivers().FindBySeason(ctx, seasonID)
	if err != nil {
		return err
	}
	c.events, err = c.r.Events().LoadBySeasonID(ctx, seasonID)
	if err != nil {
		return err
	}
	c.carModelVariants, err = c.r.Cars().CarModelVariants().LoadBySeasonID(ctx, seasonID)
	if err != nil {
		return err
	}
	c.cmvLookup = make(map[int32]*dbModels.CarModelVariant)
	for _, cmv := range c.carModelVariants {
		c.cmvLookup[cmv.ID] = cmv
	}

	c.seasonCarClasses, err = c.r.Cars().CarClasses().LoadBySeasonID(ctx, seasonID)
	if err != nil {
		return err
	}
	c.ccLookup = make(map[int32]*dbModels.CarClass)
	c.cmv2ccLookup = make(map[int32]*dbModels.CarClass)
	for _, cc := range c.seasonCarClasses {
		c.ccLookup[cc.ID] = cc
		cmvInClass, err := c.r.Cars().CarModelVariants().LoadByCarClassID(ctx, cc.ID)
		if err != nil {
			return err
		}
		for _, cmv := range cmvInClass {
			c.cmv2ccLookup[cmv.ID] = cc
		}
	}

	return nil
}

//nolint:whitespace //editor/linter issue
func (c *EntryComposer) CreateDriverLookupByIDs(
	ctx context.Context, driverIDs []int32,
) map[int32]*model.Entry {
	ret := make(map[int32]*model.Entry)
	driverLookup := lo.SliceToMap(c.drivers,
		func(item *dbModels.Driver) (int32, *dbModels.Driver) {
			return item.ID, item
		})

	for _, sd := range c.seasonDrivers {
		if _, ok := lo.Find(driverIDs,
			func(item int32) bool { return item == sd.DriverID }); !ok {
			continue
		}
		ret[sd.DriverID] = &model.Entry{
			ID:      sd.DriverID,
			Name:    driverLookup[sd.DriverID].Name,
			CarNum:  sd.CarNumber,
			Rookie:  sd.IsRookie,
			CarName: c.cmvLookup[sd.CarModelVariantID].Name,
		}
		if c.season.IsMulticlass {
			ret[sd.DriverID].CarClass = c.cmv2ccLookup[sd.CarModelVariantID].Name
		}

	}

	return ret
}

//nolint:whitespace //editor/linter issue
func (c *EntryComposer) CreateDriverLookup(
	ctx context.Context,
) map[int32]*model.Entry {
	ids := lo.Map(c.seasonDrivers, func(sd *dbModels.SeasonDriver, _ int) int32 {
		return sd.DriverID
	})
	return c.CreateDriverLookupByIDs(ctx, ids)
}

//nolint:whitespace //editor/linter issue
func (c *EntryComposer) CreateDriverLookupByTeams(
	ctx context.Context,
) map[int32]*model.Entry {
	ids := lo.Map(c.teamDrivers, func(td *dbModels.TeamDriver, _ int) int32 {
		return td.DriverID
	})
	return c.CreateDriverLookupByIDs(ctx, ids)
}

//nolint:whitespace //editor/linter issue
func (c *EntryComposer) CreateTeamLookup(
	ctx context.Context,
) map[int32]*model.Entry {
	ret := make(map[int32]*model.Entry)

	for _, team := range c.teams {
		ret[team.ID] = &model.Entry{
			ID:   team.ID,
			Name: team.Name,
		}
		if c.season.IsTeamBased {
			cmv, _ := lo.Coalesce(
				c.cmvLookup[team.CarModelVariantID.GetOrZero()],
				&dbModels.CarModelVariant{Name: "n.a."})

			cClass, _ := lo.Coalesce(
				c.cmv2ccLookup[cmv.ID],
				&dbModels.CarClass{Name: "n.a."})
			ret[team.ID].CarNum = team.CarNumber.GetOr("n.a.")
			ret[team.ID].CarName = cmv.Name
			ret[team.ID].CarClass = cClass.Name
		}
	}

	return ret
}

func (c *EntryComposer) CarClasses() []*dbModels.CarClass {
	return c.seasonCarClasses
}
