package staticbackend

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
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
		slog.Error(err.Error())
	}
}

func parseBody(body io.ReadCloser, v interface{}) error {
	defer func() { _ = body.Close() }()
	return json.NewDecoder(body).Decode(v)
}
