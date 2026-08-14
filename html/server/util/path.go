package util

import (
	"fmt"
	"net/url"
	"sync"

	"github.com/srlmgr/backend/html/server/model"
)

var (
	pathContext   model.PathContext
	pathContextMu sync.RWMutex
)

func InitPathContext(contextPath, externalURL string) {
	pathContextMu.Lock()
	defer pathContextMu.Unlock()

	pathContext = model.PathContext{
		ContextPath: contextPath,
		ExternalURL: externalURL,
	}
}

func WithBaseURLOverride(baseURL string, fn func()) {
	pathContextMu.Lock()
	saved := pathContext
	if baseURL == "" {
		pathContextMu.Unlock()
		fn()
		return
	}
	pathContext = model.PathContext{
		ContextPath: saved.ContextPath,
		ExternalURL: baseURL,
	}
	defer func() {
		pathContext = saved
		pathContextMu.Unlock()
	}()
	fn()
}

// composes an URL used for navigation on the generated pages
func ComposeNavURL(arg string) string {
	pathContextMu.RLock()
	defer pathContextMu.RUnlock()

	if pathContext.ExternalURL != "" {
		return fmt.Sprintf("%s%s", pathContext.ExternalURL, arg)
	}
	return arg
}

func HandlerURL(path string) string {
	pathContextMu.RLock()
	defer pathContextMu.RUnlock()

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
