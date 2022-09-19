package backend

import (
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/staticbackendhq/core/internal"
	"github.com/staticbackendhq/core/model"
)

type FileStore struct {
	auth model.Auth
	conf model.BaseConfig
}

func newFile(auth model.Auth, conf model.BaseConfig) FileStore {
	return FileStore{
		auth: auth,
		conf: conf,
	}
}

type SavedFile struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

func (f FileStore) Save(filename, name string, file io.ReadSeeker, size int64) (sf SavedFile, err error) {
	ext := filepath.Ext(name)

	if len(name) == 0 {
		// if no forced name is used, let's use the original name
		name = internal.CleanUpFileName(filename)
	}

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
