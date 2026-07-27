package staticbackend

import (
	"errors"
	"net/http"

	"github.com/staticbackendhq/core/backend"
	"github.com/staticbackendhq/core/database"
	"github.com/staticbackendhq/core/function"
	"github.com/staticbackendhq/core/middleware"
	"github.com/staticbackendhq/core/model"
)

type functions struct {
	dbName    string
	datastore database.Persister
}

func (f *functions) add(w http.ResponseWriter, r *http.Request) {
	conf, _, err := middleware.Extract(r, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	data := new(struct {
		ID           string `json:"id"`
		AccountID    string `json:"accountId"`
		FunctionName string `json:"name"`
		TriggerTopic string `json:"trigger"`
		Code         string `json:"code"`
		Secrets      string `json:"secrets"`
		Version      int    `json:"version"`
	})
	if err := parseBody(r.Body, data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	secrets, err := model.EncryptFunctionSecrets(data.Secrets)
	if err != nil {
		if errors.Is(err, model.ErrInvalidFunctionSecretKey) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fn := model.ExecData{
		ID:           data.ID,
		AccountID:    data.AccountID,
		FunctionName: data.FunctionName,
		TriggerTopic: data.TriggerTopic,
		Code:         data.Code,
		Secrets:      secrets,
		Version:      data.Version,
	}

	if _, err := backend.DB.AddFunction(conf.Name, fn); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := refreshFunctionTriggerCache(conf.Name, fn.TriggerTopic); err != nil {
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

	data := new(struct {
		ID      string  `json:"id"`
		Code    string  `json:"code"`
		Trigger string  `json:"trigger"`
		Secrets *string `json:"secrets"`
	})
	if err := parseBody(r.Body, data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	update := model.FunctionUpdate{
		ID:           data.ID,
		Code:         data.Code,
		TriggerTopic: data.Trigger,
	}
	if data.Secrets != nil {
		secrets, err := model.EncryptFunctionSecrets(*data.Secrets)
		if err != nil {
			if errors.Is(err, model.ErrInvalidFunctionSecretKey) {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		update.Secrets = secrets
		update.UpdateSecrets = true
	}

	if err := backend.DB.UpdateFunction(conf.Name, update); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := refreshFunctionTriggerCache(conf.Name, data.Trigger); err != nil {
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

	name := getURLPart(r.URL.Path, 3)
	fn, err := backend.DB.GetFunctionByName(conf.Name, name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := backend.DB.DeleteFunction(conf.Name, name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := refreshFunctionTriggerCache(conf.Name, fn.TriggerTopic); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func refreshFunctionTriggerCache(dbName, trigger string) error {
	funcs, err := backend.DB.ListFunctionsByTrigger(dbName, trigger)
	if err != nil {
		return err
	}

	ids := make([]string, 0, len(funcs))
	for _, fn := range funcs {
		if err := backend.Cache.SetTyped("fn_"+fn.ID, fn); err != nil {
			return err
		}
		ids = append(ids, fn.ID)
	}

	return backend.Cache.SetTyped(dbName+":"+trigger, ids)
}

func (f *functions) exec(w http.ResponseWriter, r *http.Request) {
	conf, auth, err := middleware.Extract(r, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	//TODONOW: this is not needed as only the fn name is required here
	/*var data internal.ExecData
	if err := parseBody(r.Body, &data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}*/

	functionName := getURLPart(r.URL.Path, 3)

	fn, err := backend.DB.GetFunctionForExecution(conf.Name, functionName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	env := &function.ExecutionEnvironment{
		Auth:      auth,
		BaseName:  conf.Name,
		DataStore: backend.DB,
		Search:    backend.Search,
		Volatile:  backend.Cache,
		Data:      fn,
		Email:     backend.Emailer,
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

	results, err := backend.DB.ListFunctions(conf.Name)
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

	name := getURLPart(r.URL.Path, 3)

	fn, err := backend.DB.GetFunctionByName(conf.Name, name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, fn)
}
