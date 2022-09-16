package middleware

import (
	"errors"
	"net/http"

	"github.com/staticbackendhq/core/model"
)

type ContextKey int

const (
	ContextAuth ContextKey = iota
	ContextBase
)

func Extract(r *http.Request, withAuth bool) (model.BaseConfig, model.Auth, error) {
	ctx := r.Context()
	conf, ok := ctx.Value(ContextBase).(model.BaseConfig)
	if !ok {
		return model.BaseConfig{}, model.Auth{}, errors.New("could not find config")
	}

	auth, ok := ctx.Value(ContextAuth).(model.Auth)
	if !ok && withAuth {
		return model.BaseConfig{}, model.Auth{}, errors.New("invalid StaticBackend key")
	}

	return conf, auth, nil
}
