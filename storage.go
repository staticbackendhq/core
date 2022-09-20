package staticbackend

import (
	"net/http"

	"github.com/staticbackendhq/core/backend"
	"github.com/staticbackendhq/core/middleware"
)

func upload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	conf, auth, err := middleware.Extract(r, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	file, h, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// check for file size
	// TODO: This should be based on current plan
	if h.Size/(1000*1000) > 150 {
		http.Error(w, "file size exeeded your limit", http.StatusBadRequest)
		return
	}

	name := r.Form.Get("name")

	fileSvc := backend.Storage(auth, conf)
	savedFile, err := fileSvc.Save(h.Filename, name, file, h.Size)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, savedFile)
}

func deleteFile(w http.ResponseWriter, r *http.Request) {
	conf, auth, err := middleware.Extract(r, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fileID := r.URL.Query().Get("id")

	fileSvc := backend.Storage(auth, conf)
	if err := fileSvc.Delete(fileID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, true)
}
