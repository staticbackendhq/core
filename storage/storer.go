package storage

import "github.com/staticbackendhq/core/model"

const (
	StorageProviderLocal = "local"
	StorageProviderS3    = "s3"
)

// Storer handles file saving/deleting
type Storer interface {
	// Save saves a file via a storage provider
	Save(model.UploadFileData) (string, error)
	// Delete removes a file via a storage provider
	Delete(string) error
}
