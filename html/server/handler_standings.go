package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/a-h/templ"
	"github.com/samber/lo"

	"github.com/srlmgr/backend/html/server/model"
	"github.com/srlmgr/backend/html/server/service"
	mainTempl "github.com/srlmgr/backend/html/server/templates"
	"github.com/srlmgr/backend/html/server/templates/standings"
	"github.com/srlmgr/backend/html/server/util"
	gs "github.com/srlmgr/backend/service"
	svcStandings "github.com/srlmgr/backend/service/standings"
)

type (
	standingsProcessor struct {
		s        service.Service
		r        *http.Request
		w        http.ResponseWriter
		skipMode svcStandings.SkipModeType
	}
)

func registerStandingsRoutes(mux *http.ServeMux, s service.Service) {
	mux.HandleFunc(util.GetHandlerURL("/seasons/{seasonID}"),
		func(w http.ResponseWriter, r *http.Request) {
			seasonID, err := strconv.Atoi(r.PathValue("seasonID"))
			if err != nil {
				http.NotFound(w, r)
				return
			}
			http.Redirect(
				w,
				r,
				util.HandlerURL(util.SeasonsStandingsURL(seasonID)),
				http.StatusFound,
			)
		})
	mux.HandleFunc(util.GetHandlerURL("/seasons/{seasonID}/standings"),
		func(w http.ResponseWriter, r *http.Request) {
			seasonID, err := strconv.Atoi(r.PathValue("seasonID"))
			if err != nil {
				http.NotFound(w, r)
				return
			}
			http.Redirect(
				w,
				r,
				util.HandlerURL(util.SeasonsStandingsURL(seasonID))+"/primary",
				http.StatusFound,
			)
		})
	mux.HandleFunc(util.GetHandlerURL("/seasons/{seasonID}/standings/primary"),
		handlePrimaryStandings(s, svcStandings.SkipModeAlways),
	)

	mux.HandleFunc(util.GetHandlerURL("/seasons/{seasonID}/standings/primary/noskip"),
		handlePrimaryStandings(s, svcStandings.SkipModeNever),
	)
	mux.HandleFunc(util.GetHandlerURL("/seasons/{seasonID}/standings/secondary"),
		handleSecondaryStandings(s),
	)
	mux.HandleFunc(util.GetHandlerURL("/seasons/{seasonID}/standings/primary/rookies"),
		handlePrimaryRookieStandings(s),
	)
}

//nolint:whitespace //editor/linter issue
func handlePrimaryStandings(
	s service.Service,
	skipMode svcStandings.SkipModeType,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		argSkipMode := r.URL.Query().Get("skipMode")
		if argSkipMode != "" {
			skipMode, err = svcStandings.ParseSkipMode(argSkipMode)
			if err != nil {
				http.Error(w,
					fmt.Sprintf("invalid skip mode: %v", err),
					http.StatusBadRequest)
				return
			}
		}

		p := &standingsProcessor{s: s, w: w, r: r, skipMode: skipMode}
		sData := p.process()
		if sData == nil {
			http.Error(w,
				"could not produce standings",
				http.StatusBadRequest)
			return
		}
		sData.CurrentSkipMode = argSkipMode

		sData.CurrentPath = r.URL.Path
		sData.CurrentSkipMode = argSkipMode

		var contents templ.Component
		if sData.ServiceData.Season.IsTeamBased {
			contents = standings.PrimaryTeamStandings(sData)
		} else {
			contents = standings.PrimaryDriverStandings(sData)
		}
		renderInput := standings.StandingsContent(sData, contents)
		if err := mainTempl.HTML(renderInput).Render(r.Context(), w); err != nil {
			http.Error(w,
				fmt.Sprintf("failed to render standings: %v", err),
				http.StatusInternalServerError)
			return
		}
	}
}

//nolint:whitespace //editor/linter issue
func handleSecondaryStandings(
	s service.Service,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := &standingsProcessor{s: s, w: w, r: r, skipMode: svcStandings.SkipModeNever}
		sData := p.process()
		if sData == nil {
			http.Error(w,
				"could not produce secondary standings",
				http.StatusBadRequest)
			return
		}
		sData.CurrentPath = r.URL.Path

		var contents templ.Component
		if sData.ServiceData.Season.IsTeamBased {
			contents = standings.SecondaryTeamStandings(sData)
		} else {
			contents = standings.SecondaryDriverStandings(sData)
		}

		renderInput := standings.StandingsContent(sData, contents)
		if err := mainTempl.HTML(renderInput).Render(r.Context(), w); err != nil {
			http.Error(w,
				fmt.Sprintf("failed to render standings: %v", err),
				http.StatusInternalServerError)
			return
		}
	}
}

//nolint:whitespace //editor/linter issue
func handlePrimaryRookieStandings(
	s service.Service,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := &standingsProcessor{s: s, w: w, r: r, skipMode: svcStandings.SkipModeNever}
		sData := p.process()
		if sData == nil {
			http.Error(w,
				"could not produce standings",
				http.StatusBadRequest)
			return
		}

		var contents templ.Component
		if sData.ServiceData.Season.IsTeamBased {
			sData.ServiceData.Primary = []*gs.Standing{}
			contents = standings.PrimaryTeamStandings(sData)
		} else {
			sData.ServiceData.Primary = lo.Filter(sData.ServiceData.Primary,
				func(s *gs.Standing, _ int) bool {
					return sData.PrimaryLookup[int32(s.ReferenceID)].Rookie
				})
			contents = standings.PrimaryDriverStandings(sData)
		}

		renderInput := standings.StandingsContent(sData, contents)
		if err := mainTempl.HTML(renderInput).Render(r.Context(), w); err != nil {
			http.Error(w,
				fmt.Sprintf("failed to render standings: %v", err),
				http.StatusInternalServerError)
			return
		}
	}
}

//nolint:funlen // ok here
func (p *standingsProcessor) process() *model.SeasonStandingsContainer {
	seasonID, err := strconv.Atoi(p.r.PathValue("seasonID"))
	if err != nil {
		http.Error(p.w, fmt.Sprintf("invalid season ID: %v", err), http.StatusBadRequest)
		return nil
	}
	var sData *model.SeasonStandingsContainer
	var sErr error
	if p.r.URL.Query().Get("eventID") != "" {
		eventID, err := strconv.Atoi(p.r.URL.Query().Get("eventID"))
		if err != nil {
			http.Error(p.w,
				fmt.Sprintf("invalid event ID: %v", err),
				http.StatusBadRequest)
			return nil
		}
		sData, sErr = p.s.GetEventStandings(p.r.Context(), eventID, p.skipMode)
		sData.CurrentEventID = eventID
	} else {
		sData, sErr = p.s.GetSeasonStandings(p.r.Context(), seasonID, p.skipMode)
	}
	if sErr != nil {
		http.Error(p.w,
			fmt.Sprintf("failed to load season: %v", sErr),
			http.StatusInternalServerError)
		return nil
	}

	if sData.ServiceData.Season.IsMulticlass {
		if p.r.URL.Query().Get("classID") == "" {
			http.Redirect(
				p.w,
				p.r,
				fmt.Sprintf("%s?classID=%d", p.r.URL.Path, sData.CarClasses[0].ID),
				http.StatusFound)
			return nil
		}
		classID, err := strconv.Atoi(p.r.URL.Query().Get("classID"))
		if err != nil {
			http.Error(p.w,
				fmt.Sprintf("invalid class ID: %v", err),
				http.StatusBadRequest)
			return nil
		}
		sData.ServiceData.Primary = sData.FilterByClass(
			sData.ServiceData.Primary, classID)
		sData.ServiceData.Secondary = sData.FilterByClass(
			sData.ServiceData.Secondary, classID)
		sData.CurrentClassID = classID
	}
	sData.CurrentPath = p.r.URL.Path
	sData.NavData = &myNav{
		sc:          sData.SeasonsContainer,
		season:      sData.ServiceData.Season,
		qParam:      p.r.URL.Query(),
		currentPath: p.r.URL.Path,
		carClasses:  sData.CarClasses,
	}
	return sData
}
