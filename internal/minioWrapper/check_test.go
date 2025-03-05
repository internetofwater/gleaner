package minioWrapper

import (
	"context"
	"log"
	"testing"

	minioClient "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	assert.NoError(t, err)
	url, err := minioContainer.ConnectionString(ctx)
	assert.NoError(t, err)
	mc, err := minioClient.New(url, &minioClient.Options{
		Creds:  credentials.NewStaticV4(minioContainer.Username, minioContainer.Password, ""),
		Secure: false,
	})
	assert.NoError(t, err)
	buckets, err := mc.ListBuckets(context.Background())
	assert.NoError(t, err)
	assert.Empty(t, buckets)

	client := MinioClientWrapper{Client: mc, DefaultBucket: ""}
	err = client.SetupBucket()
	require.Error(t, err) // make sure that an "" default bucket results in an error
	client = MinioClientWrapper{Client: mc, DefaultBucket: "gleanerbucket"}
	err = client.SetupBucket()
	assert.NoError(t, err)
	buckets, err = mc.ListBuckets(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, buckets[0].Name, "gleanerbucket")
}
