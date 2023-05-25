package staticbackend

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/staticbackendhq/core/backend"
)

func respond(w http.ResponseWriter, code int, v interface{}) {
	b, err := json.Marshal(v)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if _, err := w.Write(b); err != nil {
		backend.Log.Error().Err(err)
	}
}

func parseBody(body io.ReadCloser, v interface{}) error {
	defer body.Close()
	return json.NewDecoder(body).Decode(v)
}
