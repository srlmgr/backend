package authn

import (
	"errors"
	"net/http"
	"strings"
)

const (
	apiTokenHeader = "api-token"
)

type authSource int

const (
	authSourceNone authSource = iota
	authSourceAPIToken
)

type selectedCredential struct {
	source authSource
	token  string
}

func selectCredential(headers http.Header) (selectedCredential, error) {
	authorizationValue := strings.TrimSpace(headers.Get("Authorization"))
	if authorizationValue != "" {
		return selectedCredential{}, errors.New("authorization header is not supported")
	}

	apiToken := strings.TrimSpace(headers.Get(apiTokenHeader))
	hasAPIToken := apiToken != ""

	if hasAPIToken {
		return selectedCredential{source: authSourceAPIToken, token: apiToken}, nil
	}

	return selectedCredential{source: authSourceNone}, nil
}
