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
	View     string
	Subview  string // example: primary, secondary for standings, results
	Subtype  string // example: rookies in standings
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
		req, err := newSnippetRequest(r)
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

func newSnippetRequest(r *http.Request) (snippetRequest, error) {
	q := r.URL.Query()
	view := strings.TrimSpace(q.Get("view"))

	if view == "" {
		view = "standings"
	}

	seasonID, err := parseOptionalInt(q.Get("seasonID"))
	if err != nil {
		return snippetRequest{}, fmt.Errorf("invalid seasonID: %w", err)
	}
	if view != "series" && seasonID == 0 {
		return snippetRequest{}, fmt.Errorf("missing seasonID")
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
		View:     view,
		Subview:  strings.TrimSpace(q.Get("subview")),
		Subtype:  strings.TrimSpace(q.Get("subtype")),
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

	switch strings.ToLower(req.View) {
	case "dummy":
		component, err = snippetDummy(r, s, req)
	case "participants":
		component, err = snippetParticipants(r, s, req)
	case "standings":
		component, err = snippetStandings(r, s, req)

	case "results-overview", "results_overview", "resultsoverview", "overview":
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
	if req.SkipMode != "" {
		skipMode, err = svcStandings.ParseSkipMode(req.SkipMode)
		if err != nil {
			return nil, fmt.Errorf("invalid skip mode: %w", err)
		}
	}

	if req.EventID != 0 {
		data, err = s.GetEventStandings(r.Context(), req.EventID, skipMode)
	} else {
		data, err = s.GetSeasonStandings(r.Context(), req.SeasonID, skipMode)
		if req.Subtype == "rookies" {
			data.ServiceData.Primary = lo.Filter(data.ServiceData.Primary,
				func(s *gs.Standing, _ int) bool {
					return data.PrimaryLookup[int32(s.ReferenceID)].Rookie
				})
		}
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
		data.CurrentClassID = classID
	}
	data.CurrentPath = r.URL.Path
	data.CurrentSkipMode = req.SkipMode
	data.NavData = &snippetNav{
		sc:      data.SeasonsContainer,
		season:  data.ServiceData.Season,
		qParam:  r.URL.Query(),
		cmsPath: req.CMSPath,
		cmsURL:  req.CMSUrl,

		carClasses: data.CarClasses,
	}
	wrapper := func(contents templ.Component) templ.Component {
		return standings.StandingsSnippet(data, contents)
	}
	if strings.EqualFold(req.Subview, "secondary") {
		if data.ServiceData.Season.IsTeamBased {
			return wrapper(standings.SecondaryTeamStandings(data)), nil
		}
		return wrapper(standings.SecondaryDriverStandings(data)), nil
	}

	if data.ServiceData.Season.IsTeamBased {
		return wrapper(standings.PrimaryTeamStandings(data, true)), nil
	}
	return wrapper(standings.PrimaryDriverStandings(data, true)), nil
}

//nolint:whitespace //editor/linter issue
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
	}
	wrapper := func(contents templ.Component) templ.Component {
		return resultsoverview.OverviewSnippet(data, contents)
	}
	if strings.Contains(strings.ToLower(r.URL.Query().Get("mode")), "secondary") ||
		strings.Contains(strings.ToLower(req.View), "secondary") {

		return wrapper(resultsoverview.SecondaryOverview(data)), nil
	}
	return wrapper(resultsoverview.PrimaryOverview(data)), nil
}

type snippetNav struct {
	sc         *model.SeasonsContainer
	season     *dbModels.Season
	carClasses []*model.CarClass
	qParam     url.Values
	cmsPath    string
	cmsURL     string
}

var _ model.SeasonNav = (*snippetNav)(nil)

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
