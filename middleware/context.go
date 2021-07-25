package middleware

import (
	"errors"
	"net/http"
)

type ContextKey int

const (
	ContextAuth ContextKey = iota
	ContextBase
)

func Extract(r *http.Request, withAuth bool) (BaseConfig, Auth, error) {
	ctx := r.Context()
	conf, ok := ctx.Value(ContextBase).(BaseConfig)
	if !ok {
		return BaseConfig{}, Auth{}, errors.New("could not find config")
	}

	auth, ok := ctx.Value(ContextAuth).(Auth)
	if !ok && withAuth {
		return BaseConfig{}, Auth{}, errors.New("invalid StaticBackend key")
	}

	return conf, auth, nil
}
