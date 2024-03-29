// Package middleware exposes middlewares and helpers functions
// related to context get/set of DatabaseConfig and Auth.
package middleware

import (
	"errors"
	"net/http"

	"github.com/staticbackendhq/core/model"
)

type ContextKey int

// Context keys that are needed for all requests pipeline.
const (
	ContextAuth ContextKey = iota
	ContextBase
)

// Extract extracts the DatabaseConfig and Auth for the request
func Extract(r *http.Request, withAuth bool) (model.DatabaseConfig, model.Auth, error) {
	ctx := r.Context()
	conf, ok := ctx.Value(ContextBase).(model.DatabaseConfig)
	if !ok {
		return model.DatabaseConfig{}, model.Auth{}, errors.New("could not find config")
	}

	auth, ok := ctx.Value(ContextAuth).(model.Auth)
	if !ok && withAuth {
		return model.DatabaseConfig{}, model.Auth{}, errors.New("invalid StaticBackend key")
	}

	return conf, auth, nil
}
