package staticbackend

import (
	"net/http"
	"staticbackend/db"
	"staticbackend/function"
	"staticbackend/middleware"
)

type functions struct {
	base *db.Base
}

type ExecData struct {
	FunctionName string `json:"name"`
}

func (f *functions) exec(w http.ResponseWriter, r *http.Request) {
	conf, auth, err := middleware.Extract(r, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var data ExecData
	if err := parseBody(r.Body, &data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	curDB := client.Database(conf.Name)

	env := &function.ExecutionEnvironment{
		Auth: auth,
		DB:   curDB,
		Base: f.base,
	}
	if err := env.Execute(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
