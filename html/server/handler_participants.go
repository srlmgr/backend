package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/a-h/templ"

	"github.com/srlmgr/backend/html/server/model"
	"github.com/srlmgr/backend/html/server/service"
	mainTempl "github.com/srlmgr/backend/html/server/templates"
	"github.com/srlmgr/backend/html/server/templates/participants"
	"github.com/srlmgr/backend/html/server/util"
)

type (
	particiantsProcessor struct {
		s service.Service
		r *http.Request
		w http.ResponseWriter
	}
)

func registerParticipantsRoutes(mux *http.ServeMux, s service.Service) {
	mux.HandleFunc(util.GetHandlerURL("/seasons/{seasonID}/participants"),
		func(w http.ResponseWriter, r *http.Request) {
			seasonID, err := strconv.Atoi(r.PathValue("seasonID"))
			if err != nil {
				http.NotFound(w, r)
				return
			}

			target := util.HandlerURL(fmt.Sprintf("/seasons/%d/participants/primary", seasonID))

			http.Redirect(w, r, target, http.StatusFound)
		},
	)
	mux.HandleFunc(util.GetHandlerURL("/seasons/{seasonID}/participants/primary"),
		handlePrimaryParticipants(s),
	)
}

//nolint:whitespace //editor/linter issue
func handlePrimaryParticipants(
	s service.Service,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := &particiantsProcessor{s: s, w: w, r: r}
		sData := p.process()
		if sData == nil {
			http.Error(w,
				"could not produce standings",
				http.StatusBadRequest)
			return
		}

		var contents templ.Component
		if sData.Season.IsTeamBased {
			contents = participants.PrimarySeasonTeam(sData)
		} else {
			contents = participants.PrimarySeasonDriver(sData)
		}
		renderInput := participants.ParticipantsContent(sData, contents)
		if err := mainTempl.HTML(renderInput).Render(r.Context(), w); err != nil {
			http.Error(w,
				fmt.Sprintf("failed to render standings: %v", err),
				http.StatusInternalServerError)
			return
		}
	}
}

func (p *particiantsProcessor) process() *model.SeasonParticipantsContainer {
	seasonID, err := strconv.Atoi(p.r.PathValue("seasonID"))
	if err != nil {
		http.Error(p.w, fmt.Sprintf("invalid season ID: %v", err), http.StatusBadRequest)
		return nil
	}
	var sData *model.SeasonParticipantsContainer
	var sErr error
	sData, sErr = p.s.GetSeasonParticipants(p.r.Context(), seasonID)
	if sErr != nil {
		http.Error(p.w,
			fmt.Sprintf("failed to get season participants: %v", sErr),
			http.StatusInternalServerError)
		return nil
	}
	sData.NavData = &myNav{
		sc:          sData.SeasonsContainer,
		season:      sData.Season,
		qParam:      p.r.URL.Query(),
		currentPath: p.r.URL.Path,
	}
	return sData
}
