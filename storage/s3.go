package storage

import (
	"context"
	"fmt"

	"github.com/staticbackendhq/core/config"
	"github.com/staticbackendhq/core/model"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type S3 struct{}

func (S3) Save(data model.UploadFileData) (string, error) {
	ctx := context.Background()
	endpoint := config.Current.S3Endpoint
	accessKeyID := config.Current.S3AccessKey
	secretAccessKey := config.Current.S3SecretKey
	bucketName := config.Current.S3Bucket

	// Initialize minio client object.
	c, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: true,
	})
	if err != nil {
		return "", err
	}

	contentType := data.Mimetype
	if len(contentType) == 0 {
		contentType = "application/octet-stream"
	}

	opts := minio.PutObjectOptions{ContentType: contentType}
	_, err = c.PutObject(ctx, bucketName, data.FileKey, data.File, data.Size, opts)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf(
		"%s/%s",
		config.Current.S3CDNURL,
		data.FileKey,
	)

	return url, nil
}

func (S3) Delete(fileKey string) error {
	ctx := context.Background()
	endpoint := config.Current.S3Endpoint
	accessKeyID := config.Current.S3AccessKey
	secretAccessKey := config.Current.S3SecretKey

	// Initialize minio client object.
	c, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: true,
	})
	if err != nil {
		return err
	}

	return c.RemoveObject(ctx, config.Current.S3Bucket, fileKey, minio.RemoveObjectOptions{})
}
