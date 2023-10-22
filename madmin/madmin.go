package madmin

import (
	"context"
	"log"
	"os"

	"github.com/minio/madmin-go/v3"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stenstromen/miniomatic/db"
)

func Madmin(Id, RootUser, RootPassword, BucketName, AccessKey, SecretKey string) error {

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

	accountInfo, err := madminClient.AccountInfo(context.Background(), madmin.AccountOpts{})
	if err != nil {
		log.Fatalln(err)
	}

	log.Println(accountInfo.AccountName)
	log.Println(string(accountInfo.Policy))

	// Initialize standard MinIO client
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(AccessKey, SecretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatalln(err)
	}

	// Create a new bucket for the user
	location := "us-east-1" // Region; adjust as necessary

	err = minioClient.MakeBucket(context.Background(), BucketName, minio.MakeBucketOptions{Region: location})
	if err != nil {
		log.Printf("%v already exists", BucketName)
		os.Exit(0)
	}

	//log.Printf("%v created, %v created, and %v set successfully!", accountInfo.AccountName, BucketName, AccessKey)
	log.Printf("%v rootuser, %v rootpassword", RootUser, RootPassword)
	db.UpdateStatus(Id, "ready")
	return nil
}
