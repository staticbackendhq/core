package staticbackend

import (
	"bytes"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/staticbackendhq/core/extra"
	"github.com/staticbackendhq/core/internal"
	"github.com/staticbackendhq/core/middleware"
)

type extras struct{}

func (ex *extras) resizeImage(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.Header.Get("Content-Type"))
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		fmt.Println("cannot parse form")
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
	// there's a maximum of 2GB for image manupulation
	if h.Size/(1000*1000) > 2 {
		http.Error(w, "file size exeeded your limit", http.StatusBadRequest)
		return
	}

	ext := filepath.Ext(h.Filename)

	//TODO: Remove all but a-zA-Z/ from name

	name := r.Form.Get("name")
	if len(name) == 0 {
		name = randStringRunes(32)
	}

	newWidth, err := strconv.ParseFloat(r.Form.Get("width"), 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var b []byte
	buf := bytes.NewBuffer(b)

	if err := extra.ResizeImage(h.Filename, file, buf, newWidth); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fileKey := fmt.Sprintf("%s/%s/%s%s",
		auth.AccountID,
		config.Name,
		name,
		ext,
	)

	resizedBytes := buf.Bytes()
	fmt.Println("resized bytes: ", len(resizedBytes))
	upData := internal.UploadFileData{FileKey: fileKey, File: bytes.NewReader(resizedBytes)}
	url, err := storer.Save(upData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	f := internal.File{
		AccountID: auth.AccountID,
		Key:       fileKey,
		URL:       url,
		Size:      int64(len(b)),
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
