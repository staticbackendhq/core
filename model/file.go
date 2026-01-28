package model

import (
	"io"
	"time"
)

type UploadFileData struct {
	FileKey  string
	File     io.ReadSeeker
	Size     int64
	Mimetype string
}

type File struct {
	ID        string    `json:"id"`
	AccountID string    `json:"accountId"`
	Key       string    `json:"key"`
	URL       string    `json:"url"`
	Size      int64     `json:"size"`
	Uploaded  time.Time `json:"uploaded"`
}
