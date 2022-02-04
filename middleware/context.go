package middleware

import (
	"errors"
	"net/http"

	"github.com/staticbackendhq/core/internal"
)

type ContextKey int

const (
	ContextAuth ContextKey = iota
	ContextBase
)

func Extract(r *http.Request, withAuth bool) (internal.BaseConfig, internal.Auth, error) {
	ctx := r.Context()
	conf, ok := ctx.Value(ContextBase).(internal.BaseConfig)
	if !ok {
		return internal.BaseConfig{}, internal.Auth{}, errors.New("could not find config")
	}

	auth, ok := ctx.Value(ContextAuth).(internal.Auth)
	if !ok && withAuth {
		return internal.BaseConfig{}, internal.Auth{}, errors.New("invalid StaticBackend key")
	}

	return conf, auth, nil
}
