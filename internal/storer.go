package internal

import "io"

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
