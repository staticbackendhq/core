package staticbackend

import (
	"net/http"
	"strings"

	"github.com/staticbackendhq/core/backend"
	"github.com/staticbackendhq/core/middleware"
)

func submitForm(w http.ResponseWriter, r *http.Request) {
	conf, _, err := middleware.Extract(r, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	form := ""

	_, r.URL.Path = ShiftPath(r.URL.Path)
	form, r.URL.Path = ShiftPath(r.URL.Path)

	doc := make(map[string]interface{})

	//TODO: Why forms would need multiplart/form-data
	// There's no file upload available via form
	/*if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}*/

	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// if there's something in the _hp_ field, it's a bot
	if len(r.Form.Get("_hp_")) > 0 {
		http.Error(w, "invalid form field present", http.StatusBadRequest)
		return
	}

	for k, v := range r.Form {
		doc[k] = strings.Join(v, ", ")
	}

	if err := backend.DB.AddFormSubmission(conf.Name, form, doc); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, true)
}

func listForm(w http.ResponseWriter, r *http.Request) {
	conf, _, err := middleware.Extract(r, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	formName := r.URL.Query().Get("name")

	results, err := backend.DB.ListFormSubmissions(conf.Name, formName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, results)
}
