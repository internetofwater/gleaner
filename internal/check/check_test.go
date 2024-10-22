package check

import (
	"context"
	"log"
	"testing"

	minioClient "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/testcontainers/testcontainers-go"

	minio "github.com/testcontainers/testcontainers-go/modules/minio"
)

func TestConnCheck(t *testing.T) {

	ctx := context.Background()

	minioContainer, err := minio.Run(ctx, "minio/minio:latest")
	defer func() {
		if err := testcontainers.TerminateContainer(minioContainer); err != nil {
			log.Printf("failed to terminate container: %s", err)
		}
	}()
	if err != nil {
		t.Fatalf("failed to start container: %s", err)
	}
	url, err := minioContainer.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("failed to get connection string: %s", err)
	}
	mc, err := minioClient.New(url, &minioClient.Options{
		Creds:  credentials.NewStaticV4(minioContainer.Username, minioContainer.Password, ""),
		Secure: false,
	})
	if err != nil {
		t.Fatalf("Failed to create MinIO client: %v", err)
	}

	buckets, err := mc.ListBuckets(context.Background())
	if err != nil {
		t.Fatalf("List buckets failed: %v", err)
	}
	t.Log("Buckets:", buckets)
}
