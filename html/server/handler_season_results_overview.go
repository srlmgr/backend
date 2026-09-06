package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/srlmgr/backend/html/server/model"
	"github.com/srlmgr/backend/html/server/service"
	mainTempl "github.com/srlmgr/backend/html/server/templates"
	"github.com/srlmgr/backend/html/server/templates/resultsoverview"
	"github.com/srlmgr/backend/html/server/util"
)

type (
	resultsOverviewProcessor struct {
		s service.Service
		r *http.Request
		w http.ResponseWriter
	}
)

func registerSeasonResultsOverviewRoutes(mux *http.ServeMux, s service.Service) {
	mux.HandleFunc(util.GetHandlerURL("/seasons/{seasonID}/results/overview/primary"),
		handleSeasonResultsOverview(s),
	)
	mux.HandleFunc(util.GetHandlerURL("/seasons/{seasonID}/results/overview/secondary"),
		handleSeasonResultsOverviewSecondary(s),
	)
}

//nolint:whitespace,dupl //editor/linter issue
func handleSeasonResultsOverview(
	s service.Service,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := &resultsOverviewProcessor{s: s, w: w, r: r}
		sData := p.process()
		if sData == nil {
			http.Error(w,
				"could not produce results overview",
				http.StatusBadRequest)
			return
		}

		contents := resultsoverview.PrimaryOverview(sData)

		renderInput := resultsoverview.OverviewContent(sData, contents)
		if err := mainTempl.HTML(renderInput).Render(r.Context(), w); err != nil {
			http.Error(w,
				fmt.Sprintf("failed to render primary results overview: %v", err),
				http.StatusInternalServerError)
			return
		}
	}
}

//nolint:whitespace,dupl //editor/linter issue
func handleSeasonResultsOverviewSecondary(
	s service.Service,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := &resultsOverviewProcessor{s: s, w: w, r: r}
		sData := p.process()
		if sData == nil {
			http.Error(w,
				"could not produce results overview",
				http.StatusBadRequest)
			return
		}

		contents := resultsoverview.SecondaryOverview(sData)

		renderInput := resultsoverview.OverviewContent(sData, contents)
		if err := mainTempl.HTML(renderInput).Render(r.Context(), w); err != nil {
			http.Error(w,
				fmt.Sprintf("failed to render secondary results overview: %v", err),
				http.StatusInternalServerError)
			return
		}
	}
}

//nolint:funlen // lots of checks
func (p *resultsOverviewProcessor) process() *model.SeasonResultsOverviewContainer {
	seasonID, err := strconv.Atoi(p.r.PathValue("seasonID"))
	if err != nil {
		http.Error(p.w, fmt.Sprintf("invalid season ID: %v", err), http.StatusBadRequest)
		return nil
	}
	classID := 0
	season, _ := p.s.GetSeason(p.r.Context(), seasonID)
	if season.IsMulticlass {
		carClasses, err := p.s.GetSeasonCarClasses(p.r.Context(), seasonID)
		if err != nil {
			http.Error(p.w,
				fmt.Sprintf("failed to get car classes: %v", err),
				http.StatusInternalServerError)
			return nil
		}

		if p.r.URL.Query().Get("classID") == "" {
			http.Redirect(
				p.w,
				p.r,
				fmt.Sprintf("%s?classID=%d", p.r.URL.Path, carClasses[0].ID),
				http.StatusFound,
			)
			return nil
		}
		classID, err = strconv.Atoi(p.r.URL.Query().Get("classID"))
		if err != nil {
			http.Error(p.w,
				fmt.Sprintf("invalid class ID: %v", err),
				http.StatusBadRequest)
			return nil
		}
	}
	var sData *model.SeasonResultsOverviewContainer
	var sErr error
	sData, sErr = p.s.GetResultsOverview(p.r.Context(), seasonID, classID)
	if sErr != nil {
		http.Error(p.w,
			fmt.Sprintf("failed to get season results overview: %v", sErr),
			http.StatusInternalServerError)
		return nil
	}
	sData.NavData = &myNav{
		sc:          sData.SeasonsContainer,
		season:      sData.ServiceData.Season,
		qParam:      p.r.URL.Query(),
		currentPath: p.r.URL.Path,
		carClasses:  sData.CarClasses,
	}
	return sData
}
