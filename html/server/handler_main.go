package server

import (
	"net/http"
	"net/url"

	dbModels "github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/html/server/model"
	"github.com/srlmgr/backend/html/server/service"
	"github.com/srlmgr/backend/html/server/util"
)

//nolint:unparam //service may be used later
func registerMainRoutes(mux *http.ServeMux, s service.Service) {
	mux.HandleFunc("GET /",
		func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/" {
				http.NotFound(w, r)
				return
			}
			http.Redirect(
				w,
				r,
				util.HandlerURL("/serieses"),
				http.StatusFound,
			)
		})
}

type myNav struct {
	sc          *model.SeasonsContainer
	season      *dbModels.Season
	carClasses  []*model.CarClass
	qParam      url.Values
	currentPath string
}

var _ model.SeasonNav = (*myNav)(nil)

func (m *myNav) ContextPath() string {
	return contextPart
}

func (m *myNav) ExternalURL() string {
	return externalURL
}

func (m *myNav) CurrentPath() string {
	return m.currentPath
}

func (m *myNav) Season() *dbModels.Season {
	return m.season
}

func (m *myNav) Seasons() []*model.Season {
	return m.sc.Seasons
}

func (m *myNav) SeriesContainer() *model.SeriesContainer {
	return m.sc.SeriesContainer
}

func (m *myNav) CarClasses() []*model.CarClass {
	return m.carClasses
}

func (m *myNav) QueryParam() url.Values {
	return m.qParam
}
