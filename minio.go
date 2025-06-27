package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type MinioClient struct {
	client     *minio.Client
	bucketName string
}

func NewMinioClient() *MinioClient {
	endpoint := viper.GetString("minio.endpoint")
	
	// Environment variables take precedence over config file
	accessKey := os.Getenv("MINIO_ACCESS_KEY")
	if accessKey == "" {
		accessKey = viper.GetString("minio.access_key")
	}
	
	secretKey := os.Getenv("MINIO_SECRET_KEY")
	if secretKey == "" {
		secretKey = viper.GetString("minio.secret_key")
	}
	
	useSSL := viper.GetBool("minio.use_ssl")
	bucketName := viper.GetString("minio.bucket_name")

	if endpoint == "" {
		log.Fatal("MinIO endpoint not configured")
	}
	if accessKey == "" {
		log.Fatal("MinIO access key not configured")
	}
	if secretKey == "" {
		log.Fatal("MinIO secret key not configured")
	}
	if bucketName == "" {
		log.Fatal("MinIO bucket name not configured")
	}

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatal("Failed to create MinIO client:", err)
	}

	return &MinioClient{
		client:     minioClient,
		bucketName: bucketName,
	}
}

func (m *MinioClient) UploadFile(filePath, objectName string) error {
	log.Infof("Uploading file %s to MinIO as %s", filePath, objectName)

	ctx := context.Background()

	exists, err := m.client.BucketExists(ctx, m.bucketName)
	if err != nil {
		return fmt.Errorf("failed to check if bucket exists: %w", err)
	}

	if !exists {
		log.Infof("Creating bucket %s", m.bucketName)
		err = m.client.MakeBucket(ctx, m.bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	fileStat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	contentType := "application/octet-stream"
	if filepath.Ext(filePath) == ".gpg" {
		contentType = "application/pgp-encrypted"
	}

	uploadInfo, err := m.client.PutObject(ctx, m.bucketName, objectName, file, fileStat.Size(), minio.PutObjectOptions{
		ContentType: contentType,
		UserMetadata: map[string]string{
			"original-filename": filepath.Base(filePath),
			"backup-tool":      "k8s-ceph-backup",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}

	log.Infof("Successfully uploaded %s to MinIO bucket %s (ETag: %s, Size: %d bytes)", 
		objectName, m.bucketName, uploadInfo.ETag, uploadInfo.Size)

	return nil
}

func (m *MinioClient) DownloadFile(objectName, filePath string) error {
	log.Infof("Downloading object %s from MinIO to %s", objectName, filePath)

	ctx := context.Background()

	object, err := m.client.GetObject(ctx, m.bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to get object: %w", err)
	}
	defer object.Close()

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer file.Close()

	stat, err := object.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat object: %w", err)
	}

	if _, err := file.ReadFrom(object); err != nil {
		return fmt.Errorf("failed to download object: %w", err)
	}

	log.Infof("Successfully downloaded %s from MinIO (Size: %d bytes)", objectName, stat.Size)

	return nil
}

func (m *MinioClient) ListObjects(prefix string) ([]string, error) {
	log.Debugf("Listing objects in bucket %s with prefix %s", m.bucketName, prefix)

	ctx := context.Background()

	var objects []string

	objectCh := m.client.ListObjects(ctx, m.bucketName, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	for object := range objectCh {
		if object.Err != nil {
			return nil, fmt.Errorf("error listing objects: %w", object.Err)
		}
		objects = append(objects, object.Key)
	}

	log.Debugf("Found %d objects with prefix %s", len(objects), prefix)

	return objects, nil
}

func (m *MinioClient) DeleteObject(objectName string) error {
	log.Infof("Deleting object %s from MinIO", objectName)

	ctx := context.Background()

	err := m.client.RemoveObject(ctx, m.bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	log.Infof("Successfully deleted object %s from MinIO", objectName)

	return nil
}

func (m *MinioClient) ObjectExists(objectName string) (bool, error) {
	log.Debugf("Checking if object %s exists in MinIO", objectName)

	ctx := context.Background()

	_, err := m.client.StatObject(ctx, m.bucketName, objectName, minio.StatObjectOptions{})
	if err != nil {
		if errResponse := minio.ToErrorResponse(err); errResponse.Code == "NoSuchKey" {
			return false, nil
		}
		return false, fmt.Errorf("failed to check object existence: %w", err)
	}

	return true, nil
}