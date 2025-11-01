package main

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {

	// Use the SDK to create a s3.PresignClient with s3.NewPresignClient
	presignClnt := s3.NewPresignClient(s3Client)

	// struct for passing the bucket/key into the request
	objInpt := s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	}

	// create the http request
	httpReq, err := presignClnt.PresignGetObject(context.Background(), &objInpt, s3.WithPresignExpires(expireTime))
	if err != nil {
		return "", err
	}

	// return the url
	return httpReq.URL, nil

}
