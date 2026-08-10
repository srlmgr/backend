package util

import (
	"fmt"
	"net/url"

	"github.com/srlmgr/backend/html/server/model"
)

var pathContext model.PathContext

func InitPathContext(contextPath, externalURL string) {
	pathContext = model.PathContext{
		ContextPath: contextPath,
		ExternalURL: externalURL,
	}
}

// composes an URL used for navigation on the generated pages
func ComposeNavURL(arg string) string {
	if pathContext.ExternalURL != "" {
		return fmt.Sprintf("%s%s", pathContext.ExternalURL, arg)
	}
	return arg
}

func HandlerURL(path string) string {
	if pathContext.ContextPath != "" {
		return fmt.Sprintf("%s%s", pathContext.ContextPath, path)
	}
	return path
}

func GetHandlerURL(path string) string {
	return "GET " + HandlerURL(path)
}

func SeasonsURL(id int) string {
	return HandlerURL(fmt.Sprintf("/seasons/%d", id))
}

func SeasonsStandingsURL(id int) string {
	return HandlerURL(fmt.Sprintf("/seasons/%d/standings", id))
}

func EnsureQueryParam(qParam url.Values, key, value string) url.Values {
	work := qParam
	if qParam == nil {
		work = url.Values{}
	}
	work.Set(key, value)
	return work
}
