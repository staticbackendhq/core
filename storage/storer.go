package storage

import "github.com/staticbackendhq/core/model"

const (
	StorageProviderLocal = "local"
	StorageProviderS3    = "s3"
)

type Storer interface {
	Save(model.UploadFileData) (string, error)
	Delete(string) error
}
