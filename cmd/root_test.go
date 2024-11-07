package cmd

import (
	"context"
	"io"
	"strings"
	"testing"

	"gleaner/test_helpers"

	sitemaps "gleaner/internal/summoner/sitemaps"

	minioClient "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/testcontainers/testcontainers-go"
)

// Test gleaner when run on a fresh s3 bucket
func TestRootE2E(t *testing.T) {

	ctx := context.Background()

	minioContainer, err := test_helpers.MinioRun(ctx, "minio/minio:latest")
	require.NoError(t, err)

	url, _, err := test_helpers.ConnectionStrings(ctx, minioContainer)
	require.NoError(t, err)

	gleanerCliArgs := &GleanerCliArgs{
		AccessKey:    minioContainer.Username,
		SecretKey:    minioContainer.Password,
		Address:      strings.Split(url, ":")[0],
		Port:         strings.Split(url, ":")[1],
		Source:       "mainstems",
		Config:       "../test_helpers/just_mainstems.yaml",
		SetupBuckets: true,
	}

	defer testcontainers.TerminateContainer(minioContainer)

	if err := Gleaner(gleanerCliArgs); err != nil {
		t.Fatal(err)
	}

	mc, err := minioClient.New(url, &minioClient.Options{
		Creds:  credentials.NewStaticV4(minioContainer.Username, minioContainer.Password, ""),
		Secure: false,
	})
	require.NoError(t, err)

	buckets, err := mc.ListBuckets(context.Background())
	require.NoError(t, err)
	require.Equal(t, buckets[0].Name, "gleanerbucket")

	// After the first run, only one org metadata should be present
	orgsInfo, orgs, err := test_helpers.GetGleanerBucketObjects(mc, "orgs/")

	require.NoError(t, err)
	require.Equal(t, 1, len(orgs)) // should only have one org since we only crawled one site
	require.Equal(t, 1, len(orgsInfo))
	orgDataBytes, err := io.ReadAll(orgs[0])
	require.NoError(t, err)
	orgData1 := string(orgDataBytes)

	// After first run, we should have as many objects as sites in the sitemap
	sumInfo, summoned, err := test_helpers.GetGleanerBucketObjects(mc, "summoned/")
	require.NoError(t, err)
	sitesOnWebpage, err := sitemaps.ParseSitemap("https://pids.geoconnex.dev/sitemap/ref/mainstems/mainstems__0.xml")
	require.NoError(t, err)
	require.Equal(t, len(sitesOnWebpage.URL), len(summoned))
	require.Equal(t, len(sitesOnWebpage.URL), len(sumInfo))

	// Run it again
	if err := Gleaner(gleanerCliArgs); err != nil {
		t.Fatal(err)
	}

	// Check that after the second run, the org metadata should be unchanged since it is with the same data
	orgsInfo2, orgs2, err := test_helpers.GetGleanerBucketObjects(mc, "orgs/")
	require.NoError(t, err)
	assert.Equal(t, len(orgsInfo2), len(orgsInfo))
	assert.Equal(t, len(orgs2), len(orgsInfo))
	orgDataBytes2, err := io.ReadAll(orgs2[0])
	orgData2 := string(orgDataBytes2)
	require.NoError(t, err)
	result := test_helpers.AssertLinesMatchDisregardingOrder(orgData1, orgData2)
	assert.True(t, result)

	// Check that the we have exactly as many sites in the sitemap
	sumInfo2, summoned2, err := test_helpers.GetGleanerBucketObjects(mc, "summoned/")
	require.NoError(t, err)
	assert.Equal(t, len(sitesOnWebpage.URL), len(summoned2))
	assert.Equal(t, len(sitesOnWebpage.URL), len(sumInfo2))
}

func TestGeoconnexPids(t *testing.T) {
	ctx := context.Background()

	minioContainer, err := test_helpers.MinioRun(ctx, "minio/minio:latest")
	if err != nil {
		t.Fatalf("failed to start container: %s", err)
	}
	url, _, err := test_helpers.ConnectionStrings(ctx, minioContainer)
	require.NoError(t, err)

	gleanerCliArgs := &GleanerCliArgs{
		AccessKey:    minioContainer.Username,
		SecretKey:    minioContainer.Password,
		Address:      strings.Split(url, ":")[0],
		Port:         strings.Split(url, ":")[1],
		Config:       "../test_helpers/geoconnex-pids.yaml",
		SetupBuckets: true,
	}

	defer testcontainers.TerminateContainer(minioContainer)

	if err := Gleaner(gleanerCliArgs); err != nil {
		t.Fatal(err)
	}
	mc, err := minioClient.New(url, &minioClient.Options{
		Creds:  credentials.NewStaticV4(minioContainer.Username, minioContainer.Password, ""),
		Secure: false,
	})
	require.NoError(t, err)

	assertions := func() {
		test_helpers.AssertObjectCount(t, mc, "orgs/", 5)
		test_helpers.AssertObjectCount(t, mc, "summoned/cdss0/", 30)
		test_helpers.AssertObjectCount(t, mc, "prov/dams0/", 45)
		test_helpers.AssertObjectCount(t, mc, "prov/nmwdist0/", 266)
		test_helpers.AssertObjectCount(t, mc, "prov/refgages0/", 330)
		test_helpers.AssertObjectCount(t, mc, "prov/refmainstems/", 66)
		test_helpers.AssertObjectCount(t, mc, "summoned/dams0/", 45)
		test_helpers.AssertObjectCount(t, mc, "summoned/nmwdist0/", 265)
		test_helpers.AssertObjectCount(t, mc, "summoned/refgages0/", 330)
		test_helpers.AssertObjectCount(t, mc, "summoned/refmainstems/", 66)
	}
	assertions()

	if err := Gleaner(gleanerCliArgs); err != nil {
		t.Fatal(err)
	}

	// Asserts it is idempotent
	assertions()

}

func TestSitemapWithDeadLink(t *testing.T) {

	ctx := context.Background()

	minioContainer, err := test_helpers.MinioRun(ctx, "minio/minio:latest")
	require.NoError(t, err)

	url, _, err := test_helpers.ConnectionStrings(ctx, minioContainer)
	require.NoError(t, err)

	gleanerCliArgs := &GleanerCliArgs{
		AccessKey:    minioContainer.Username,
		SecretKey:    minioContainer.Password,
		Address:      strings.Split(url, ":")[0],
		Port:         strings.Split(url, ":")[1],
		Source:       "DUMMY",
		Config:       "../test_helpers/sample_configs/invalidSitemap.yaml",
		SetupBuckets: true,
	}

	defer testcontainers.TerminateContainer(minioContainer)

	if err := Gleaner(gleanerCliArgs); err != nil {
		t.Fatal(err)
	}

	mc, err := minioClient.New(url, &minioClient.Options{
		Creds:  credentials.NewStaticV4(minioContainer.Username, minioContainer.Password, ""),
		Secure: false,
	})
	require.NoError(t, err)

	// After the first run, only one org metadata should be present
	orgsInfo, orgs, err := test_helpers.GetGleanerBucketObjects(mc, "orgs/")

	require.NoError(t, err)
	require.Equal(t, 1, len(orgs))
	require.Equal(t, 1, len(orgsInfo))
	require.NoError(t, err)

	const prefixToGetAllItems = ""
	_, allItems, err := test_helpers.GetGleanerBucketObjects(mc, prefixToGetAllItems)
	require.NoError(t, err)
	require.Equal(t, 1, len(allItems)) // should only have one org since we only crawled one site
}
