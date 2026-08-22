package util

import (
	"fmt"
	"net/url"
	"slices"

	"github.com/srlmgr/backend/html/server/model"
)

var pathContext model.PathContext

func InitPathContext(contextPath, externalURL string) {
	pathContext = model.PathContext{
		ContextPath: contextPath,
		ExternalURL: externalURL,
	}
}

func AdjustedSnippetURL(path string, nav model.CommonNav) string {
	work := HandlerURL(path)
	if nav.ContextPath() != "" {
		work = nav.ContextPath()
	}
	if nav.ExternalURL() != "" {
		return fmt.Sprintf("%s%s", nav.ExternalURL(), work)
	}
	return work
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
	work := make(url.Values, len(qParam))

	for k, v := range qParam {
		work[k] = slices.Clone(v)
	}

	work.Set(key, value)
	return work
}
