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

func getGleanerBucketObjects(mc *minioClient.Client, subDir string) ([]minioClient.ObjectInfo, []*minioClient.Object, error) {
	var metadata []minioClient.ObjectInfo
	var objects []*minioClient.Object
	objectCh := mc.ListObjects(context.Background(), "gleanerbucket", minioClient.ListObjectsOptions{Recursive: true, Prefix: subDir})

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

	buckets, err := mc.ListBuckets(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, buckets[0].Name, "gleanerbucket")

	// After the first run, only one org metadata should be present
	orgsInfo, orgs, err := getGleanerBucketObjects(mc, "orgs/")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(orgs))
	assert.Equal(t, 1, len(orgsInfo))
	orgDataBytes, err := io.ReadAll(orgs[0])
	assert.NoError(t, err)
	orgData1 := string(orgDataBytes)

	// After first run, we should have as many objects as sites in the sitemap
	sumInfo, summoned, err := getGleanerBucketObjects(mc, "summoned/")
	assert.NoError(t, err)
	const numberOfSitesInref_hu02_hu02__0Sitemap = 22
	assert.Equal(t, numberOfSitesInref_hu02_hu02__0Sitemap, len(summoned))
	assert.Equal(t, numberOfSitesInref_hu02_hu02__0Sitemap, len(sumInfo))

	// Run it again
	if err := Gleaner(); err != nil {
		t.Fatal(err)
	}

	// Check that after the second run, the org metadata should be unchanged since it is with the same data
	orgsInfo2, orgs2, err := getGleanerBucketObjects(mc, "orgs/")
	assert.NoError(t, err)
	assert.Equal(t, len(orgsInfo2), len(orgsInfo))
	assert.Equal(t, len(orgs2), len(orgsInfo))
	orgDataBytes2, err := io.ReadAll(orgs2[0])
	orgData2 := string(orgDataBytes2)
	assert.NoError(t, err)
	assert.Equal(t, orgData1, orgData2)

	// Check that the we have twice as many sites in the sitemap
	sumInfo2, summoned2, err := getGleanerBucketObjects(mc, "summoned/")
	assert.NoError(t, err)
	assert.Equal(t, (2*numberOfSitesInref_hu02_hu02__0Sitemap)-4, len(summoned2))
	assert.Equal(t, (2*numberOfSitesInref_hu02_hu02__0Sitemap)-4, len(sumInfo2))
}
