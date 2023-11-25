package madmin

import (
	"context"
	"log"
	"os"

	"github.com/minio/madmin-go/v3"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stenstromen/miniomatic/db"
	"github.com/stenstromen/miniomatic/model"
)

func Madmin(creds model.Credentials, BucketName, AccessKey, SecretKey string) error {
	Id, RootUser, RootPassword := creds.RandNum, creds.RootUser, creds.RootPassword
	endpoint := Id + "." + os.Getenv("WILDCARD_DOMAIN")
	useSSL := true

	// Initialize MinIO admin client
	madminClient, err := madmin.New(endpoint, RootUser, RootPassword, useSSL)
	if err != nil {
		log.Fatalln(err)
	}

	// User creation
	err = madminClient.AddUser(context.Background(), AccessKey, SecretKey)
	if err != nil {
		log.Fatalln(err)
	}
	err = madminClient.SetPolicy(context.Background(), "readwrite", AccessKey, false)
	if err != nil {
		log.Fatalln(err)
	}

	// Initialize standard MinIO client
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(AccessKey, SecretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatalln(err)
	}

	// Create a new bucket for the user
	location := "eu-north-1"

	err = minioClient.MakeBucket(context.Background(), BucketName, minio.MakeBucketOptions{Region: location})
	if err != nil {
		log.Printf("%v already exists", BucketName)
		os.Exit(0)
	}

	db.UpdateStatus(Id, "ready")
	return nil
}
