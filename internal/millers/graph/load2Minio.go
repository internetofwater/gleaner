package graph

import (
	"bytes"
	"context"

	log "github.com/sirupsen/logrus"

	minio "github.com/minio/minio-go/v7"
)

// LoadToMinio loads jsonld into the specified bucket
func LoadToMinio(jsonld, bucketName, objectName string, mc *minio.Client) (int64, error) {

	// set up some elements for PutObject
	b := bytes.NewBufferString(jsonld)
	usermeta := make(map[string]string)

	// Upload the zip file with FPutObject
	n, err := mc.PutObject(context.Background(), bucketName, objectName, b, int64(b.Len()), minio.PutObjectOptions{ContentType: "application/ld+json", UserMetadata: usermeta})
	if err != nil {
		log.Error(bucketName, "/", objectName, "error", err)
		return 0, err
	}

	log.Trace("Uploaded Bucket:", bucketName, "File:", objectName, "Size", n)

	return n.Size, nil
}
