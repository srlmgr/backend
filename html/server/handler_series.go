package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/srlmgr/backend/html/server/service"
	mainTempl "github.com/srlmgr/backend/html/server/templates"
	seasonsTempl "github.com/srlmgr/backend/html/server/templates/seasons"
	seriesTempl "github.com/srlmgr/backend/html/server/templates/series"
	"github.com/srlmgr/backend/html/server/util"
)

func registerSeriesesRoutes(mux *http.ServeMux, s service.Service) {
	// TODO: think about redirect to first season of series
	mux.HandleFunc(util.GetHandlerURL("/serieses"),
		handleSerieses(s),
	)
	mux.HandleFunc(util.GetHandlerURL("/serieses/{seriesID}"),
		handleSeries(s),
	)
}

//nolint:whitespace //editor/linter issue
func handleSerieses(
	s service.Service,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		seriesContainer, err := s.GetSeriesList(r.Context())
		if err != nil {
			http.Error(w,
				fmt.Sprintf("failed to get series list: %v", err),
				http.StatusInternalServerError)
			return
		}
		if len(seriesContainer.Serieses) > 0 {
			http.Redirect(
				w,
				r,
				util.HandlerURL(fmt.Sprintf("/serieses/%d", seriesContainer.Serieses[0].ID)),
				http.StatusFound,
			)
			return
		}

		sContents := seriesTempl.SeriesNav(seriesContainer)
		if err := mainTempl.HTML(sContents).Render(r.Context(), w); err != nil {
			http.Error(w,
				fmt.Sprintf("failed to render standings: %v", err),
				http.StatusInternalServerError)
			return
		}
	}
}

//nolint:whitespace //editor/linter issue
func handleSeries(
	s service.Service,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		seriesID, err := strconv.Atoi(r.PathValue("seriesID"))
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid series ID: %v", err), http.StatusBadRequest)
			return
		}
		seasonContainer, err := s.GetSeasonList(r.Context(), seriesID)
		if err != nil {
			http.Error(w,
				fmt.Sprintf("failed to get season list: %v", err),
				http.StatusInternalServerError)
			return
		}
		data := &myNav{sc: seasonContainer}

		sContents := seasonsTempl.SeasonsContents(data)
		if err := mainTempl.HTML(sContents).Render(r.Context(), w); err != nil {
			http.Error(w,
				fmt.Sprintf("failed to render seasons: %v", err),
				http.StatusInternalServerError)
			return
		}
	}
}
