package service

import (
	"context"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/service/standings"
	"github.com/srlmgr/backend/service/summary"
)

type (
	StandingsType int
	Standing      struct {
		Type            StandingsType
		ReferenceID     int
		CarClassID      int
		EventID         int
		Data            *standings.StandingData
		DroppedEventIDs []int
	}
	StandingsContainer struct {
		Season    *models.Season
		Primary   []*Standing
		Secondary []*Standing
	}
	SummaryContainer struct {
		Season             *models.Season
		Events             []*summary.EventSummary
		PrimarySummaries   []*summary.SummaryEntry
		SecondarySummaries []*summary.SummaryEntry
	}
	Service interface {
		StandingsService() StandingsService
		ParticipantsService() ParticipantsService
		BookingsService() BookingsService
		SummaryService() SummaryService
	}
	StandingsService interface {
		// gets standings for the last season event
		GetSeasonStandings(
			ctx context.Context,
			seasonID int,
			skipMode standings.SkipModeType) (*StandingsContainer, error)
		// gets standings for a specific event
		// (all events up to (including) the specified event are included)
		GetEventStandings(
			ctx context.Context,
			eventID int,
			skipMode standings.SkipModeType) (*StandingsContainer, error)
	}
	BookingsService interface{} // TODO: may be not needed
	SummaryService  interface {
		GetSeasonSummary(
			ctx context.Context,
			seasonID int) (*SummaryContainer, error)
		GetEventSummary(
			ctx context.Context,
			eventID int) (*summary.EventSummary, error)
	}
	ParticipantsService interface {
		GetDrivers(ctx context.Context, seasonID int) ([]*models.SeasonDriver, error)
		GetTeams(ctx context.Context, seasonID int) ([]*models.Team, error)
	}
)

// keep this order in sync with proto StandingsType enum
const (
	StandingsTypeUnspecified StandingsType = iota
	StandingsTypeDriver
	StandingsTypeTeam
)
