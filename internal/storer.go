package internal

import (
	"io"
	"time"
)

const (
	StorageProviderLocal = "local"
	StorageProviderS3    = "s3"
)

type UploadFileData struct {
	FileKey string
	File    io.ReadSeeker
}

type Storer interface {
	Save(UploadFileData) (string, error)
	Delete(string) error
}

type File struct {
	ID        string    `json:"id"`
	AccountID string    `json:"accountId"`
	Key       string    `json:"key"`
	URL       string    `json:"url"`
	Size      int64     `json:"size"`
	Uploaded  time.Time `json:"uploaded"`
}
