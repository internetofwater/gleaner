package cmd

import (
	"context"
	"io"
	"strings"
	"testing"

	minioClient "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/minio"
)

func getGleanerObjects(mc *minioClient.Client, prefix string) ([]minioClient.ObjectInfo, []*minioClient.Object, error) {
	var metadata []minioClient.ObjectInfo
	var objects []*minioClient.Object
	objectCh := mc.ListObjects(context.Background(), "gleanerbucket", minioClient.ListObjectsOptions{Recursive: true, Prefix: prefix})

	for object := range objectCh {
		metadata = append(metadata, object)
		obj, err := mc.GetObject(context.Background(), "gleanerbucket", object.Key, minioClient.GetObjectOptions{})
		if err != nil {
			return nil, nil, err
		}
		objects = append(objects, obj)
	}

	return metadata, objects, nil
}

// Test gleaner when run on a fresh s3 bucket
func TestRootE2E(t *testing.T) {

	ctx := context.Background()

	minioContainer, err := minio.Run(ctx, "minio/minio:latest")
	if err != nil {
		t.Fatalf("failed to start container: %s", err)
	}
	url, err := minioContainer.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("failed to get connection string: %s", err)
	}

	accessKeyVal = minioContainer.Username
	secretKeyVal = minioContainer.Password
	addressVal = strings.Split(url, ":")[0]
	portVal = strings.Split(url, ":")[1]
	sourceVal = "ref_hu02_hu02__0"
	configVal = "../test/gleanerconfig.yaml"
	setupBucketsVal = true

	defer func() {
		if err := testcontainers.TerminateContainer(minioContainer); err != nil {
			t.Errorf("failed to terminate container: %s", err)
		}
	}()
	assert.NoError(t, err)

	if err := Gleaner(); err != nil {
		t.Fatal(err)
	}

	mc, err := minioClient.New(url, &minioClient.Options{
		Creds:  credentials.NewStaticV4(minioContainer.Username, minioContainer.Password, ""),
		Secure: false,
	})
	assert.NoError(t, err)

	t.Run("Contains proper buckets", func(t *testing.T) {
		buckets, err := mc.ListBuckets(context.Background())
		if err != nil {
			t.Fatalf("List buckets failed: %v", err)
		}
		assert.Equal(t, buckets[0].Name, "gleanerbucket")
	})

	objectInfo, objects, err := getGleanerObjects(mc, "orgs/")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(objects))
	assert.Equal(t, 1, len(objectInfo))

	provDataBytes, err := io.ReadAll(objects[0])
	assert.NoError(t, err)
	provData := string(provDataBytes)

	// Run it again
	if err := Gleaner(); err != nil {
		t.Fatal(err)
	}

	// Check that the data is the same
	secondRunobjectInfo, secondRunObjects, err := getGleanerObjects(mc, "orgs/")
	assert.NoError(t, err)
	assert.Equal(t, len(secondRunobjectInfo), len(objectInfo))
	assert.Equal(t, len(secondRunObjects), len(objects))
	provDataBytes2, err := io.ReadAll(secondRunObjects[0])
	provData2 := string(provDataBytes2)
	assert.NoError(t, err)
	assert.Equal(t, provData, provData2)
}
