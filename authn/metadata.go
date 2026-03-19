package authn

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

const (
	authorizationHeader = "Authorization"
	apiTokenHeader      = "api-token"
)

type authSource int

const (
	authSourceNone authSource = iota
	authSourceJWT
	authSourceAPIToken
)

type selectedCredential struct {
	source authSource
	token  string
}

func selectCredential(headers http.Header) (selectedCredential, error) {
	bearer, hasBearer, err := extractBearerToken(headers)
	if err != nil {
		return selectedCredential{}, err
	}

	apiToken := strings.TrimSpace(headers.Get(apiTokenHeader))
	hasAPIToken := apiToken != ""

	if hasBearer && hasAPIToken {
		return selectedCredential{}, errors.New(
			"both authorization and api-token were provided",
		)
	}

	if hasBearer {
		return selectedCredential{source: authSourceJWT, token: bearer}, nil
	}

	if hasAPIToken {
		return selectedCredential{source: authSourceAPIToken, token: apiToken}, nil
	}

	return selectedCredential{source: authSourceNone}, nil
}

func extractBearerToken(headers http.Header) (string, bool, error) {
	value := strings.TrimSpace(headers.Get(authorizationHeader))
	if value == "" {
		return "", false, nil
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(value, prefix) {
		return "", false, fmt.Errorf("authorization header must use bearer scheme")
	}

	token := strings.TrimSpace(strings.TrimPrefix(value, prefix))
	if token == "" {
		return "", false, fmt.Errorf("bearer token is empty")
	}

	return token, true, nil
}
