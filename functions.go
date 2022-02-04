package staticbackend

import (
	"net/http"

	"github.com/staticbackendhq/core/db"
	"github.com/staticbackendhq/core/function"
	"github.com/staticbackendhq/core/middleware"
)

type functions struct {
	base *db.Base
}

func (f *functions) add(w http.ResponseWriter, r *http.Request) {
	conf, _, err := middleware.Extract(r, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var data function.ExecData
	if err := parseBody(r.Body, &data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	curDB := client.Database(conf.Name)

	if _, err := function.Add(curDB, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (f *functions) update(w http.ResponseWriter, r *http.Request) {
	conf, _, err := middleware.Extract(r, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	curDB := client.Database(conf.Name)

	data := new(struct {
		ID      string `json:"id"`
		Code    string `json:"code"`
		Trigger string `json:"trigger"`
	})
	if err := parseBody(r.Body, &data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := function.Update(curDB, data.ID, data.Code, data.Trigger); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (f *functions) del(w http.ResponseWriter, r *http.Request) {
	conf, _, err := middleware.Extract(r, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	curDB := client.Database(conf.Name)

	name := getURLPart(r.URL.Path, 3)
	if err := function.Delete(curDB, name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := function.Delete(curDB, name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (f *functions) exec(w http.ResponseWriter, r *http.Request) {
	conf, auth, err := middleware.Extract(r, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var data function.ExecData
	if err := parseBody(r.Body, &data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	curDB := client.Database(conf.Name)

	fn, err := function.GetForExecution(curDB, data.FunctionName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	env := &function.ExecutionEnvironment{
		Auth: auth,
		DB:   curDB,
		Base: f.base,
		Data: fn,
	}

	if err := env.Execute(r); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (f *functions) list(w http.ResponseWriter, r *http.Request) {
	conf, _, err := middleware.Extract(r, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	curDB := client.Database(conf.Name)

	results, err := function.List(curDB)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, results)
}

func (f *functions) info(w http.ResponseWriter, r *http.Request) {
	conf, _, err := middleware.Extract(r, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	curDB := client.Database(conf.Name)

	name := getURLPart(r.URL.Path, 3)

	fn, err := function.GetByName(curDB, name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, fn)
}
