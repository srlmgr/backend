package model

import (
	"net/url"
	"time"

	dbModels "github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/service"
)

type (
	CarClass struct {
		ID   int
		Name string
	}
	Event struct {
		ID   int
		Name string
		Date time.Time
	}
	SeriesContainer struct {
		Serieses []*Series
	}
	Series struct {
		ID   int
		Name string
	}
	SeasonsContainer struct {
		SeriesContainer *SeriesContainer
		Seasons         []*Season
	}
	Season struct {
		ID   int
		Name string
	}

	SeasonStandingsContainer struct {
		SeasonsContainer *SeasonsContainer
		ServiceData      service.StandingsContainer
		BaseURL          string
		CurrentPath      string
		CurrentClassID   int
		CurrentEventID   int
		CurrentSkipMode  string
		CarClasses       []*CarClass
		PrimaryLookup    map[int32]string
		SecondaryLookup  map[int32]string
		Events           []*Event
		NavData          SeasonNav
	}
	Participant interface {
		CarNumber() string
		Name() string
		CarClass() string
		CarName() string
	}
	SeasonParticipantsContainer struct {
		SeasonsContainer      *SeasonsContainer
		Season                *dbModels.Season
		Drivers               []*dbModels.SeasonDriver
		Teams                 []*dbModels.Team
		PrimaryParticipants   []Participant
		SecondaryParticipants []Participant
		PrimaryLookup         map[int32]string
		SecondaryLookup       map[int32]string
		NavData               SeasonNav
	}
	SeasonResultsOverviewContainer struct {
		SeasonsContainer      *SeasonsContainer
		ServiceData           *service.SummaryContainer
		PrimaryLookup         map[int32]string
		SecondaryLookup       map[int32]string
		Events                []*Event
		CarClasses            []*CarClass
		PrimaryMatrixLookup   map[IDTuple]int
		SecondaryMatrixLookup map[IDTuple]int
		NavData               SeasonNav
	}
)

// lookup helper
type (
	IDTuple struct {
		ReferenceID int
		// SubID can be eventID, raceID, or other depending on context
		SubID int // find better name
	}
)

// Definitions used for navigation
type (
	PathContext struct {
		ContextPath string
		ExternalURL string
	}
	CommonNav interface {
		ExternalURL() string // no ending slash, e.g. "https://srlmgr.com"
		ContextPath() string // includes leading slash, e.g. "/vrdb"
		QueryParam() url.Values
		// full path (including ExternalURL and ContextPath)
		// may be used in templ when endpoint stays the same and query param changes
		// (for example: switch between car classes)
		CurrentPath() string
	}
	SeasonNav interface {
		CommonNav
		Seasons() []*Season
		Season() *dbModels.Season
		CarClasses() []*CarClass
		SeriesContainer() *SeriesContainer
	}
	SeasonNavParam interface {
		NavParam() SeasonNav
	}
)

var (
	_ SeasonNavParam = (*SeasonParticipantsContainer)(nil)
	_ SeasonNavParam = (*SeasonResultsOverviewContainer)(nil)
	_ SeasonNavParam = (*SeasonStandingsContainer)(nil)
)

func (s *SeasonParticipantsContainer) NavParam() SeasonNav {
	return s.NavData
}

func (s *SeasonResultsOverviewContainer) NavParam() SeasonNav {
	return s.NavData
}

func (s *SeasonStandingsContainer) NavParam() SeasonNav {
	return s.NavData
}

func (s *SeasonStandingsContainer) ResolvePrimary(id int) string {
	return s.PrimaryLookup[int32(id)]
}

func (s *SeasonStandingsContainer) ResolveSecondary(id int) string {
	return s.SecondaryLookup[int32(id)]
}

//nolint:whitespace //editor/linter issue
func (s *SeasonStandingsContainer) FilterByClass(
	data []*service.Standing, classID int,
) []*service.Standing {
	var filtered []*service.Standing
	for _, s := range data {
		if s.CarClassID == classID {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

func (s *SeasonParticipantsContainer) ResolvePrimary(id int) string {
	return s.PrimaryLookup[int32(id)]
}

func (s *SeasonParticipantsContainer) ResolveSecondary(id int) string {
	return s.SecondaryLookup[int32(id)]
}

func (s *SeasonResultsOverviewContainer) ResolvePrimary(id int) string {
	return s.PrimaryLookup[int32(id)]
}

func (s *SeasonResultsOverviewContainer) ResolveSecondary(id int) string {
	return s.SecondaryLookup[int32(id)]
}

func (s *SeasonResultsOverviewContainer) PrimaryMatrix(referenceID, eventID int) int {
	return s.PrimaryMatrixLookup[IDTuple{ReferenceID: referenceID, SubID: eventID}]
}

func (s *SeasonResultsOverviewContainer) SecondaryMatrix(referenceID, eventID int) int {
	return s.SecondaryMatrixLookup[IDTuple{ReferenceID: referenceID, SubID: eventID}]
}
