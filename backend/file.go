package backend

import (
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/staticbackendhq/core/internal"
	"github.com/staticbackendhq/core/model"
)

// FileStore exposes file functions
type FileStore struct {
	auth model.Auth
	conf model.DatabaseConfig
}

func newFile(auth model.Auth, conf model.DatabaseConfig) FileStore {
	return FileStore{
		auth: auth,
		conf: conf,
	}
}

// SavedFile when a file is saved it has an ID and an URL
type SavedFile struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

// Save saves a file content to the file storage (Storer interface) and to the
// database
func (f FileStore) Save(filename, name string, file io.ReadSeeker, size int64) (sf SavedFile, err error) {
	ext := filepath.Ext(name)

	if len(name) == 0 {
		// if no forced name is used, let's use the original file name
		name = internal.CleanUpFileName(filename)
	}

	// add random char to prevent duplicate key
	name += "_" + internal.RandStringRunes(16)

	fileKey := fmt.Sprintf("%s/%s/%s%s",
		f.conf.Name,
		f.auth.AccountID,
		name,
		ext,
	)

	upData := model.UploadFileData{FileKey: fileKey, File: file}
	url, err := Filestore.Save(upData)
	if err != nil {
		return
	}

	sbFile := model.File{
		AccountID: f.auth.AccountID,
		Key:       fileKey,
		URL:       url,
		Size:      size,
		Uploaded:  time.Now(),
	}

	newID, err := DB.AddFile(f.conf.Name, sbFile)
	if err != nil {
		return
	}

	sf.ID = newID
	sf.URL = url

	return
}

// Delete removes a file from storage and database
func (f FileStore) Delete(fileID string) error {
	file, err := DB.GetFileByID(f.conf.Name, fileID)
	if err != nil {
		return err
	}

	fileKey := file.Key

	if err := Filestore.Delete(fileKey); err != nil {
		return err
	}

	if err := DB.DeleteFile(f.conf.Name, file.ID); err != nil {
		return err
	}
	return nil
}
