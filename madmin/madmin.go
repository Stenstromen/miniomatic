package madmin

import (
	"context"
	"log"
	"os"

	"github.com/minio/madmin-go/v3"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func Madmin() error {

	endpoint := "localhost:9000"
	accessKey := "user"
	secretKey := "password"
	useSSL := false

	// Initialize MinIO admin client
	madminClient, err := madmin.New(endpoint, accessKey, secretKey, useSSL)
	if err != nil {
		log.Fatalln(err)
	}

	// User creation
	newUserAccessKey := "newUserAccessKey"
	newUserSecretKey := "newUserSecretKey"
	err = madminClient.AddUser(context.Background(), newUserAccessKey, newUserSecretKey)
	if err != nil {
		log.Fatalln(err)
	}
	err = madminClient.SetPolicy(context.Background(), "readwrite", newUserAccessKey, false)
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
		Creds:  credentials.NewStaticV4(newUserAccessKey, newUserSecretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatalln(err)
	}

	// Create a new bucket for the user
	bucketName := "lol"     // Name of the new bucket
	location := "us-east-1" // Region; adjust as necessary

	err = minioClient.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{Region: location})
	if err != nil {
		log.Printf("%v already exists", bucketName)
		os.Exit(0)
	}

	log.Printf("%v created, %v created, and %v set successfully!", accountInfo.AccountName, bucketName, newUserAccessKey)
	return nil
}
