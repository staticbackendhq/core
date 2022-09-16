package storage

import (
	"fmt"

	"github.com/staticbackendhq/core/config"
	"github.com/staticbackendhq/core/model"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type S3 struct{}

func (S3) Save(data model.UploadFileData) (string, error) {
	sess, err := session.NewSession(&aws.Config{Region: aws.String("ca-central-1")})
	if err != nil {
		return "", err
	}

	svc := s3.New(sess)
	obj := &s3.PutObjectInput{}
	obj.Body = data.File
	obj.ACL = aws.String(s3.ObjectCannedACLPublicRead)
	obj.Bucket = aws.String(config.Current.AWSS3Bucket)
	obj.Key = aws.String(data.FileKey)

	if _, err := svc.PutObject(obj); err != nil {
		return "", err
	}

	url := fmt.Sprintf(
		"%s/%s",
		config.Current.AWSCDNURL,
		data.FileKey,
	)

	return url, nil
}

func (S3) Delete(fileKey string) error {
	sess, err := session.NewSession(&aws.Config{Region: aws.String("ca-central-1")})
	if err != nil {
		return err
	}

	svc := s3.New(sess)
	obj := &s3.DeleteObjectInput{
		Bucket: aws.String(config.Current.AWSS3Bucket),
		Key:    aws.String(fileKey),
	}
	if _, err := svc.DeleteObject(obj); err != nil {
		return err
	}

	return nil
}
