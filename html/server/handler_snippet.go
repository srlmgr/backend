package server

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"github.com/samber/lo"

	dbModels "github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/html/server/model"
	"github.com/srlmgr/backend/html/server/service"
	mainTempl "github.com/srlmgr/backend/html/server/templates"
	"github.com/srlmgr/backend/html/server/templates/participants"
	"github.com/srlmgr/backend/html/server/templates/resultsoverview"
	"github.com/srlmgr/backend/html/server/templates/seasons"
	"github.com/srlmgr/backend/html/server/templates/standings"
	"github.com/srlmgr/backend/html/server/util"
	gs "github.com/srlmgr/backend/service"
	svcStandings "github.com/srlmgr/backend/service/standings"
)

type snippetRequest struct {
	View     model.ViewType
	Subview  model.SubViewType // example: primary, secondary for standings, results
	Subtype  string            // example: rookies in standings
	SeriesID int
	SeasonID int
	EventID  int
	ClassID  int
	SkipMode string
	CMSPath  string
	CMSUrl   string
}

func registerSnippetRoutes(mux *http.ServeMux, s service.Service) {
	mux.HandleFunc(util.GetHandlerURL("/snippet"), handleSnippet(s))
}

func handleSnippet(s service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := newSnippetRequest(r, s)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := renderSnippet(w, r, s, &req); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

//nolint:funlen // lots of decision branches based on query parameters
func newSnippetRequest(r *http.Request, s service.Service) (snippetRequest, error) {
	q := r.URL.Query()
	view := model.ParseViewType(strings.TrimSpace(q.Get("view")))

	if view == "" {
		view = model.ViewPrimary
	}

	seriesID, err := parseOptionalInt(q.Get("seriesID"))
	if err != nil {
		return snippetRequest{}, fmt.Errorf("invalid seriesID: %w", err)
	}

	seasonID, err := parseOptionalInt(q.Get("seasonID"))
	if err != nil {
		return snippetRequest{}, fmt.Errorf("invalid seasonID: %w", err)
	}

	if seasonID == 0 && seriesID != 0 {
		sc, sErr := s.GetSeasonList(r.Context(), seriesID)
		if sErr != nil {
			return snippetRequest{}, fmt.Errorf(
				"failed to get season list for seriesID %d: %w",
				seriesID,
				sErr,
			)
		}
		if len(sc.Seasons) > 0 {
			seasonID = sc.Seasons[len(sc.Seasons)-1].ID
		}
	}
	eventID, err := parseOptionalInt(q.Get("eventID"))
	if err != nil {
		return snippetRequest{}, fmt.Errorf("invalid eventID: %w", err)
	}
	classID, err := parseOptionalInt(q.Get("classID"))
	if err != nil {
		return snippetRequest{}, fmt.Errorf("invalid classID: %w", err)
	}

	cmsPath := strings.TrimSpace(q.Get("cmsPath"))
	cmsURL := strings.TrimSpace(q.Get("cmsUrl"))

	if cmsPath == "" {
		cmsPath = strings.TrimSpace(r.Header.Get("X-CMS-Base-Path"))
	}
	if cmsURL == "" {
		cmsURL = strings.TrimSpace(r.Header.Get("X-CMS-BaseURL"))
	}

	return snippetRequest{
		View:    view,
		Subview: model.ParseSubViewType(q.Get("subview")),

		SeriesID: seriesID,
		SeasonID: seasonID,
		EventID:  eventID,
		ClassID:  classID,
		SkipMode: strings.TrimSpace(q.Get("skipMode")),
		CMSPath:  cmsPath,
		CMSUrl:   cmsURL,
	}, nil
}

func parseOptionalInt(raw string) (int, error) {
	if raw == "" {
		return 0, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, err
	}
	return v, nil
}

//nolint:whitespace //editor/linter issue
func renderSnippet(
	w http.ResponseWriter,
	r *http.Request,
	s service.Service,
	req *snippetRequest,
) error {
	var component templ.Component
	var err error

	switch req.View {
	case model.ViewDummy:
		component, err = snippetDummy(r, s, req)
	case model.ViewParticipants:
		component, err = snippetParticipants(r, s, req)
	case model.ViewPrimary, model.ViewSecondary:
		component, err = snippetStandings(r, s, req)

	case model.ViewPrimaryOverview, model.ViewSecondaryOverview:
		component, err = snippetResultsOverview(r, s, req)
	default:
		err = fmt.Errorf("unsupported snippet view %q", req.View)
	}

	if err != nil {
		return err
	}
	if component == nil {
		return fmt.Errorf("snippet render produced no content")
	}

	return mainTempl.SnippetFrame(component).Render(r.Context(), w)
}

//nolint:whitespace //editor/linter issue
func snippetDummy(
	r *http.Request,
	s service.Service,
	req *snippetRequest,
) (templ.Component, error) {
	seasonID := req.SeasonID
	if seasonID == 0 {
		return nil, fmt.Errorf("missing seasonID")
	}

	data, err := s.GetSeasonParticipants(r.Context(), seasonID)
	if err != nil {
		return nil, fmt.Errorf("get season participants: %w", err)
	}
	data.NavData = &snippetNav{
		sc:      data.SeasonsContainer,
		season:  data.Season,
		qParam:  r.URL.Query(),
		cmsPath: req.CMSPath,
		cmsURL:  req.CMSUrl,
		navValues: &snippetNavComponent{
			r: req,
		},
	}

	return seasons.SnippetSeasonsMenu(data.NavData), nil
}

//nolint:whitespace //editor/linter issue
func snippetParticipants(
	r *http.Request,
	s service.Service,
	req *snippetRequest,
) (templ.Component, error) {
	seasonID := req.SeasonID
	if seasonID == 0 {
		return nil, fmt.Errorf("missing seasonID")
	}

	data, err := s.GetSeasonParticipants(r.Context(), seasonID)
	if err != nil {
		return nil, fmt.Errorf("get season participants: %w", err)
	}
	data.NavData = &snippetNav{
		sc:      data.SeasonsContainer,
		season:  data.Season,
		qParam:  r.URL.Query(),
		cmsPath: req.CMSPath,
		cmsURL:  req.CMSUrl,
		navValues: &snippetNavComponent{
			r: req,
		},
	}
	var content templ.Component

	if data.Season.IsTeamBased {
		content = participants.PrimarySeasonTeam(data)
	} else {
		content = participants.PrimarySeasonDriver(data)
	}
	return participants.ParticipantsSnippet(data, content), nil
}

//nolint:whitespace,funlen //editor/linter issue
func snippetStandings(
	r *http.Request,
	s service.Service,
	req *snippetRequest,
) (templ.Component, error) {
	var data *model.SeasonStandingsContainer
	var err error

	skipMode := svcStandings.SkipModeAlways
	switch req.Subview {
	case model.SubViewPrimSkip:
		skipMode = svcStandings.SkipModeAlways
	case model.SubViewPrimNoSkip:
		skipMode = svcStandings.SkipModeNever
	case model.SubViewPrimRookies:
		skipMode = svcStandings.SkipModeNever
	}

	if req.EventID != 0 {
		data, err = s.GetEventStandings(r.Context(), req.EventID, skipMode)
	} else {
		data, err = s.GetSeasonStandings(r.Context(), req.SeasonID, skipMode)
		if req.Subview == model.SubViewPrimRookies {
			data.ServiceData.Primary = lo.Filter(data.ServiceData.Primary,
				func(s *gs.Standing, _ int) bool {
					x, ok := data.PrimaryLookup[int32(s.ReferenceID)]
					return ok && x.Rookie
				})
		}
		req.EventID = data.Events[len(data.Events)-1].ID
	}
	if err != nil {
		return nil, fmt.Errorf("load standings: %w", err)
	}

	if data.ServiceData.Season.IsMulticlass {
		classID := req.ClassID
		if classID == 0 {
			classID = data.CarClasses[0].ID
		}
		data.ServiceData.Primary = data.FilterByClass(data.ServiceData.Primary, classID)
		data.ServiceData.Secondary = data.FilterByClass(data.ServiceData.Secondary, classID)
	}

	data.NavData = &snippetNav{
		sc:      data.SeasonsContainer,
		season:  data.ServiceData.Season,
		qParam:  r.URL.Query(),
		cmsPath: req.CMSPath,
		cmsURL:  req.CMSUrl,

		carClasses: data.CarClasses,
		navValues: &snippetNavComponent{
			r: req,
		},
	}
	wrapper := func(contents templ.Component) templ.Component {
		return standings.StandingsSnippet(data, contents)
	}
	//nolint:exhaustive // we handle all known views, default covers the rest
	switch req.View {
	case model.ViewSecondary:
		if data.ServiceData.Season.IsTeamBased {
			return wrapper(standings.SecondaryTeamStandings(data)), nil
		}
		return wrapper(standings.SecondaryDriverStandings(data)), nil
	default:
		if data.ServiceData.Season.IsTeamBased {
			return wrapper(standings.PrimaryTeamStandings(data, true)), nil
		}
		return wrapper(standings.PrimaryDriverStandings(data, true)), nil

	}
}

//nolint:whitespace,funlen //editor/linter issue
func snippetResultsOverview(
	r *http.Request,
	s service.Service,
	req *snippetRequest,
) (templ.Component, error) {
	season, err := s.GetSeason(r.Context(), req.SeasonID)
	if err != nil {
		return nil, fmt.Errorf("get season: %w", err)
	}

	classID := req.ClassID
	if season.IsMulticlass && classID == 0 {
		classes, clErr := s.GetSeasonCarClasses(r.Context(), req.SeasonID)
		if clErr != nil {
			return nil, fmt.Errorf("get car classes: %w", clErr)
		}
		if len(classes) == 0 {
			return nil, fmt.Errorf("no car classes found for season %d", req.SeasonID)
		}
		classID = classes[0].ID
	}

	data, err := s.GetResultsOverview(r.Context(), req.SeasonID, classID)
	if err != nil {
		return nil, fmt.Errorf("load results overview: %w", err)
	}
	data.NavData = &snippetNav{
		sc:      data.SeasonsContainer,
		season:  data.ServiceData.Season,
		qParam:  r.URL.Query(),
		cmsPath: req.CMSPath,
		cmsURL:  req.CMSUrl,

		carClasses: data.CarClasses,
		navValues: &snippetNavComponent{
			r: req,
		},
	}
	wrapper := func(contents templ.Component) templ.Component {
		return resultsoverview.OverviewSnippet(data, contents)
	}
	//nolint:exhaustive // we handle all known views, default covers the rest
	switch req.View {
	case model.ViewSecondaryOverview:
		return wrapper(resultsoverview.SecondaryOverview(data)), nil
	case model.ViewPrimaryOverview:
		return wrapper(resultsoverview.PrimaryOverview(data)), nil
	default:
		return wrapper(resultsoverview.PrimaryOverview(data)), nil
	}
}

type snippetNav struct {
	sc         *model.SeasonsContainer
	season     *dbModels.Season
	carClasses []*model.CarClass
	qParam     url.Values
	cmsPath    string
	cmsURL     string
	navValues  *snippetNavComponent
}
type snippetNavComponent struct {
	r *snippetRequest
}

var (
	_ model.SeasonNav        = (*snippetNav)(nil)
	_ model.CurrentNavValues = (*snippetNavComponent)(nil)
)

// snippetNav starts here
func (m *snippetNav) ContextPath() string {
	return m.cmsPath
}

func (m *snippetNav) ExternalURL() string {
	return m.cmsURL
}

func (m *snippetNav) CurrentPath() string {
	return ""
}

func (m *snippetNav) Season() *dbModels.Season {
	return m.season
}

func (m *snippetNav) Seasons() []*model.Season {
	return m.sc.Seasons
}

func (m *snippetNav) SeriesContainer() *model.SeriesContainer {
	return m.sc.SeriesContainer
}

func (m *snippetNav) CarClasses() []*model.CarClass {
	return m.carClasses
}

func (m *snippetNav) QueryParam() url.Values {
	return m.qParam
}

func (m *snippetNav) NavValues() model.CurrentNavValues {
	return m.navValues
}

// snippetNavComponent starts here
func (m *snippetNavComponent) SeriesID() int {
	return m.r.SeriesID
}

func (m *snippetNavComponent) SeasonID() int {
	return m.r.SeasonID
}

func (m *snippetNavComponent) CarClassID() int {
	return m.r.ClassID
}

func (m *snippetNavComponent) EventID() int {
	return m.r.EventID
}

func (m *snippetNavComponent) View() model.ViewType {
	return m.r.View
}

func (m *snippetNavComponent) SubView() model.SubViewType {
	return m.r.Subview
}
