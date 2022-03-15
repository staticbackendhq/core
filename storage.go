package staticbackend

import (
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/staticbackendhq/core/internal"
	"github.com/staticbackendhq/core/middleware"
)

func upload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	config, auth, err := middleware.Extract(r, true)
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

	ext := filepath.Ext(h.Filename)

	//TODO: Remove all but a-zA-Z/ from name

	name := r.Form.Get("name")
	if len(name) == 0 {
		name = randStringRunes(32)
	}

	fileKey := fmt.Sprintf("%s/%s/%s%s",
		config.Name,
		auth.AccountID,
		name,
		ext,
	)

	upData := internal.UploadFileData{FileKey: fileKey, File: file}
	url, err := storer.Save(upData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	f := internal.File{
		AccountID: auth.AccountID,
		Key:       fileKey,
		URL:       url,
		Size:      h.Size,
		Uploaded:  time.Now(),
	}

	newID, err := datastore.AddFile(config.Name, f)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := new(struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	})
	data.ID = newID
	data.URL = url

	respond(w, http.StatusOK, data)
}

func deleteFile(w http.ResponseWriter, r *http.Request) {
	config, _, err := middleware.Extract(r, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fileID := r.URL.Query().Get("id")
	f, err := datastore.GetFileByID(config.Name, fileID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fileKey := f.Key

	if err := storer.Delete(fileKey); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := datastore.DeleteFile(config.Name, f.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, true)
}
