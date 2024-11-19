package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gleaner/internal/projectpath"
	sitemaps "gleaner/internal/summoner/sitemaps"
	"gleaner/test_helpers"

	log "github.com/sirupsen/logrus"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/testcontainers/testcontainers-go"
)

// Test gleaner's e2e for a single source in a fresh s3 bucket
func TestRootE2E(t *testing.T) {

	minioHelper, err := test_helpers.NewMinioHandle("minio/minio:latest")
	require.NoError(t, err)
	client := minioHelper.Client
	url, _, err := minioHelper.ConnectionStrings()
	require.NoError(t, err)

	gleanerCliArgs := &GleanerCliArgs{
		AccessKey:    minioHelper.Container.Username,
		SecretKey:    minioHelper.Container.Password,
		Address:      strings.Split(url, ":")[0],
		Port:         strings.Split(url, ":")[1],
		Source:       "mainstems",
		Config:       "../test_helpers/sample_configs/justMainstems.yaml",
		SetupBuckets: true,
	}
	log.Info("gleanerCliArgs: ", gleanerCliArgs)

	defer testcontainers.TerminateContainer(minioHelper.Container)

	err = Gleaner(gleanerCliArgs)
	require.NoError(t, err)

	buckets, err := minioHelper.Client.ListBuckets(context.Background())
	require.NoError(t, err)
	require.Equal(t, buckets[0].Name, "gleanerbucket")

	// After the first run, only one org metadata should be present
	orgsInfo, orgs, err := test_helpers.GetGleanerBucketObjects(client, "orgs/")

	require.NoError(t, err)
	require.Equal(t, 1, len(orgs)) // should only have one org since we only crawled one site
	require.Equal(t, 1, len(orgsInfo))
	orgFirstFileBytes, err := io.ReadAll(orgs[0])
	require.NoError(t, err)
	orgFirstFileName := orgsInfo[0].Key
	orgData1 := string(orgFirstFileBytes)

	// After first run, we should have as many objects as sites in the sitemap
	sumInfo, summoned, err := test_helpers.GetGleanerBucketObjects(client, "summoned/")
	require.NoError(t, err)
	sitesOnWebpage, err := sitemaps.ParseSitemap("https://pids.geoconnex.dev/sitemap/ref/mainstems/mainstems__0.xml")
	require.NoError(t, err)
	require.Equal(t, len(sitesOnWebpage.URL), len(summoned))
	require.Equal(t, len(sitesOnWebpage.URL), len(sumInfo))

	// Run it again
	err = Gleaner(gleanerCliArgs)
	require.NoError(t, err)

	// Check that after the second run, the org metadata should be unchanged since the orgs have not changed
	orgsInfo2, orgs2, err := test_helpers.GetGleanerBucketObjects(client, "orgs/")
	require.NoError(t, err)
	assert.Equal(t, len(orgsInfo2), len(orgsInfo))
	assert.Equal(t, len(orgs2), len(orgsInfo))
	orgFirstFileBytes2, err := io.ReadAll(orgs2[0])
	orgData2 := string(orgFirstFileBytes2)
	require.NoError(t, err)
	result := test_helpers.AssertLinesMatchDisregardingOrder(orgData1, orgData2)
	assert.True(t, result)

	// Check that the hash which is used to generate a particular file continues to exist after
	// the second runs (i.e. the file was not removed)
	test_helpers.RequireFilenameExists(t, orgsInfo2, orgFirstFileName)
	// Check that the orgs file has updated metadata even though the content inside is the same
	oldFirstFileDate := orgsInfo[0].LastModified
	test_helpers.RequireFileWasModified(t, orgsInfo2, orgFirstFileName, oldFirstFileDate)

	// Check that after the second run, we still have exactly as many objects in the summoned bucket as sites in the sitemap
	sumInfo2, summoned2, err := test_helpers.GetGleanerBucketObjects(client, "summoned/")
	require.NoError(t, err)
	assert.Equal(t, len(sitesOnWebpage.URL), len(summoned2))
	assert.Equal(t, len(sitesOnWebpage.URL), len(sumInfo2))

}

// Test gleaner's e2e for the entire geoconnex pids sitemap
func TestGeoconnexPids(t *testing.T) {

	minioHandle, err := test_helpers.NewMinioHandle("minio/minio:latest")
	require.NoError(t, err)

	url, _, err := minioHandle.ConnectionStrings()
	require.NoError(t, err)

	gleanerCliArgs := &GleanerCliArgs{
		AccessKey:    minioHandle.Container.Username,
		SecretKey:    minioHandle.Container.Password,
		Address:      strings.Split(url, ":")[0],
		Port:         strings.Split(url, ":")[1],
		Config:       "../test_helpers/sample_configs/geoconnex-pids.yaml",
		SetupBuckets: true,
	}

	defer testcontainers.TerminateContainer(minioHandle.Container)

	err = Gleaner(gleanerCliArgs)

	require.NoError(t, err)

	mc := minioHandle.Client

	assertCounts := func() {
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
	assertCounts()

	err = Gleaner(gleanerCliArgs)
	assert.NoError(t, err)

	// Asserts it is idempotent
	assertCounts()

}

// An organization with a broken sitemap will still be added to the orgs bucket
// since orgs are done before sitemap summoning
func TestSitemapWithDeadLink(t *testing.T) {

	minioHandle, err := test_helpers.NewMinioHandle("minio/minio:latest")
	require.NoError(t, err)

	url, _, err := minioHandle.ConnectionStrings()
	require.NoError(t, err)

	gleanerCliArgs := &GleanerCliArgs{
		AccessKey:    minioHandle.Container.Username,
		SecretKey:    minioHandle.Container.Password,
		Address:      strings.Split(url, ":")[0],
		Port:         strings.Split(url, ":")[1],
		Source:       "DUMMY",
		Config:       "../test_helpers/sample_configs/invalidSitemap.yaml",
		SetupBuckets: true,
	}

	defer testcontainers.TerminateContainer(minioHandle.Container)

	err = Gleaner(gleanerCliArgs)
	require.NoError(t, err)

	// After the first run, only one org metadata file should be present
	_, orgs, err := test_helpers.GetGleanerBucketObjects(minioHandle.Client, "orgs/")
	require.NoError(t, err)
	require.Equal(t, 1, len(orgs))

	const prefixToGetAllItems = "" // If we don't specify a subdir, we get everything
	_, allItems, err := test_helpers.GetGleanerBucketObjects(minioHandle.Client, prefixToGetAllItems)
	require.NoError(t, err)
	require.Equal(t, len(orgs), len(allItems)) // should only have one org since we only crawled one site
}

// We can crawl an entire config by omitting the source field
func TestEntireConfigWithoutSingleSource(t *testing.T) {

	minioHandle, err := test_helpers.NewMinioHandle("minio/minio:latest")
	require.NoError(t, err)

	url, _, err := minioHandle.ConnectionStrings()
	require.NoError(t, err)

	hu02CliArgs := &GleanerCliArgs{
		AccessKey:    minioHandle.Container.Username,
		SecretKey:    minioHandle.Container.Password,
		Address:      strings.Split(url, ":")[0],
		Port:         strings.Split(url, ":")[1],
		Config:       "../test_helpers/sample_configs/justHu02.yaml",
		SetupBuckets: true,
	}

	err = Gleaner(hu02CliArgs)
	require.NoError(t, err)

	_, summoned, err := test_helpers.GetGleanerBucketObjects(minioHandle.Client, "summoned/")
	require.NoError(t, err)
	sitesOnWebpage, err := sitemaps.ParseSitemap("https://geoconnex.us/sitemap/ref/hu02/hu02__0.xml")
	require.NoError(t, err)
	require.Equal(t, len(sitesOnWebpage.URL), len(summoned))
}

// If we crawl URLs with one config, then change the config to have new URLs
// the s3 bucket will contain both the old and new crawls
func TestCrawlsAreAdditive(t *testing.T) {

	minioHandle, err := test_helpers.NewMinioHandle("minio/minio:latest")
	require.NoError(t, err)

	url, _, err := minioHandle.ConnectionStrings()
	require.NoError(t, err)

	mainstemCliArgs := &GleanerCliArgs{
		AccessKey:    minioHandle.Container.Username,
		SecretKey:    minioHandle.Container.Password,
		Address:      strings.Split(url, ":")[0],
		Port:         strings.Split(url, ":")[1],
		Config:       "../test_helpers/sample_configs/justMainstems.yaml",
		SetupBuckets: true,
	}

	defer testcontainers.TerminateContainer(minioHandle.Container)

	err = Gleaner(mainstemCliArgs)
	require.NoError(t, err)

	_, summoned, err := test_helpers.GetGleanerBucketObjects(minioHandle.Client, "summoned/")
	require.NoError(t, err)
	mainstemWebpage, err := sitemaps.ParseSitemap("https://pids.geoconnex.dev/sitemap/ref/mainstems/mainstems__0.xml")
	require.NoError(t, err)
	require.Equal(t, len(mainstemWebpage.URL), len(summoned))

	hu02CliArgs := &GleanerCliArgs{
		AccessKey:    minioHandle.Container.Username,
		SecretKey:    minioHandle.Container.Password,
		Address:      strings.Split(url, ":")[0],
		Port:         strings.Split(url, ":")[1],
		Config:       "../test_helpers/sample_configs/justHu02.yaml",
		SetupBuckets: true,
	}

	err = Gleaner(hu02CliArgs)
	require.NoError(t, err)

	_, summoned, err = test_helpers.GetGleanerBucketObjects(minioHandle.Client, "summoned/")
	require.NoError(t, err)
	hu02Webpage, err := sitemaps.ParseSitemap("https://geoconnex.us/sitemap/ref/hu02/hu02__0.xml")
	require.NoError(t, err)

	totalURLsOnSourceWebpages := len(hu02Webpage.URL) + len(mainstemWebpage.URL)
	require.Equal(t, totalURLsOnSourceWebpages, len(summoned))

}

// If we crawl a valid sitemap, but it then becomes invalid, the old nq files
// should still be present in the s3 bucket
func TestConfigValidThenInvalid(t *testing.T) {
	minioHandle, err := test_helpers.NewMinioHandle("minio/minio:latest")
	require.NoError(t, err)

	url, _, err := minioHandle.ConnectionStrings()
	require.NoError(t, err)

	gleanerCliArgs := &GleanerCliArgs{
		AccessKey:    minioHandle.Container.Username,
		SecretKey:    minioHandle.Container.Password,
		Address:      strings.Split(url, ":")[0],
		Port:         strings.Split(url, ":")[1],
		Config:       "../test_helpers/sample_configs/justMainstems.yaml",
		SetupBuckets: true,
	}

	defer testcontainers.TerminateContainer(minioHandle.Container)

	err = Gleaner(gleanerCliArgs)
	require.NoError(t, err)

	summonedInfo, summoned, err := test_helpers.GetGleanerBucketObjects(minioHandle.Client, "summoned/")

	require.NoError(t, err)
	mainstemWebpage, err := sitemaps.ParseSitemap("https://pids.geoconnex.dev/sitemap/ref/mainstems/mainstems__0.xml")
	require.NoError(t, err)
	require.Equal(t, len(mainstemWebpage.URL), len(summoned))

	// change the config to have invalid urls
	gleanerCliArgs = &GleanerCliArgs{
		AccessKey:    minioHandle.Container.Username,
		SecretKey:    minioHandle.Container.Password,
		Address:      strings.Split(url, ":")[0],
		Port:         strings.Split(url, ":")[1],
		Config:       "../test_helpers/sample_configs/justmainstemsInvalid.yaml",
		SetupBuckets: true,
	}

	err = Gleaner(gleanerCliArgs)
	require.NoError(t, err)

	summonedInfo2, summoned2, err := test_helpers.GetGleanerBucketObjects(minioHandle.Client, "summoned/")
	require.NoError(t, err)
	urlsListedInOriginalSitemap := len(mainstemWebpage.URL)
	// there should be no additional urls added
	require.Equal(t, urlsListedInOriginalSitemap, len(summoned2))

	requireSameDates := true
	requireSizeChecks := true
	res, msg := test_helpers.SameObjects(t, summonedInfo, summonedInfo2, requireSameDates, requireSizeChecks)
	require.True(t, res, msg)
}

// Check what happens if we crawl an entire sitemap and then
// the next time we go to the sitemap, it no longer contains some sources
func TestFullThenAbbreviated(t *testing.T) {
	minioHandle, err := test_helpers.NewMinioHandle("minio/minio:latest")
	require.NoError(t, err)

	url, _, err := minioHandle.ConnectionStrings()
	require.NoError(t, err)

	gleanerCliArgs := &GleanerCliArgs{
		AccessKey:    minioHandle.Container.Username,
		SecretKey:    minioHandle.Container.Password,
		Address:      strings.Split(url, ":")[0],
		Port:         strings.Split(url, ":")[1],
		Config:       "../test_helpers/sample_configs/justMainstems.yaml",
		SetupBuckets: true,
	}

	defer testcontainers.TerminateContainer(minioHandle.Container)

	// Run gleaner with the entire
	err = Gleaner(gleanerCliArgs)
	require.NoError(t, err)

	sumInfo1, summoned1, err := test_helpers.GetGleanerBucketObjects(minioHandle.Client, "summoned/")
	require.NoError(t, err)
	mainstemWebpage, err := sitemaps.ParseSitemap("https://pids.geoconnex.dev/sitemap/ref/mainstems/mainstems__0.xml")
	require.NoError(t, err)
	require.Equal(t, len(mainstemWebpage.URL), len(summoned1))

	sampleConfDir := filepath.Join(projectpath.Root, "test_helpers", "sample_configs")
	confToAppendTo := "justMainstemsLocalEndpoint.yaml"

	// create the config that gleaner will use to find the proper sitemap
	newConfig, err := test_helpers.NewTempConfig(confToAppendTo, sampleConfDir)
	require.NoError(t, err)
	defer os.Remove(newConfig)

	// spin up the file server for our abbreviated sitemap
	server, listener, err := test_helpers.ServeSampleConfigDir()
	assert.NoError(t, err)
	defer func() {
		server.Close()
		listener.Close()
	}()

	abbreviatedSitemap := "mainstemSitemapWithoutMost.xml"
	require.FileExists(t, filepath.Join(sampleConfDir, abbreviatedSitemap))

	newConfigEndpoint := fmt.Sprintf("http://%s/%s", listener.Addr().String(), abbreviatedSitemap)
	// assert you can ping the endpoint
	resp, err := http.Get(newConfigEndpoint)
	require.NoError(t, err, "Could not get %s", newConfigEndpoint)
	require.Equal(t, 200, resp.StatusCode, "Wrong error code for %s", newConfigEndpoint)

	test_helpers.MutateConfigSourceUrl(newConfig, 0, newConfigEndpoint)

	gleanerCliArgs = &GleanerCliArgs{
		AccessKey:    minioHandle.Container.Username,
		SecretKey:    minioHandle.Container.Password,
		Address:      strings.Split(url, ":")[0],
		Port:         strings.Split(url, ":")[1],
		Config:       newConfig,
		SetupBuckets: true,
	}

	err = Gleaner(gleanerCliArgs)
	require.NoError(t, err)

	sumInfo2, summoned2, err := test_helpers.GetGleanerBucketObjects(minioHandle.Client, "summoned/")
	require.NoError(t, err)
	// the second summon should not add any new files
	require.Equal(t, len(summoned1), len(summoned2))

	// second crawl should be exactly the same since the last
	// modification date is the same
	dateChecks := true
	sizeChecks := true
	res, msg := test_helpers.SameObjects(t, sumInfo1, sumInfo2, dateChecks, sizeChecks)
	require.True(t, res, msg)

	// create another new config, but this different with different dates
	sitemapWithDifferentDates := "mainstemSitemapWithoutMostAndDifferentDate.xml"
	require.FileExists(t, filepath.Join(sampleConfDir, sitemapWithDifferentDates))

	differentDateEndpoint := fmt.Sprintf("http://%s/%s", listener.Addr().String(), sitemapWithDifferentDates)
	// assert you can ping the endpoint
	resp, err = http.Get(differentDateEndpoint)
	require.NoError(t, err, "Could not get %s", differentDateEndpoint)
	require.Equal(t, 200, resp.StatusCode, "Wrong error code for %s", differentDateEndpoint)

	test_helpers.MutateConfigSourceUrl(newConfig, 0, differentDateEndpoint)

	gleanerCliArgs = &GleanerCliArgs{
		AccessKey:    minioHandle.Container.Username,
		SecretKey:    minioHandle.Container.Password,
		Address:      strings.Split(url, ":")[0],
		Port:         strings.Split(url, ":")[1],
		Config:       newConfig,
		SetupBuckets: true,
	}

	err = Gleaner(gleanerCliArgs)
	require.NoError(t, err)

	sumInfo3, summoned3, err := test_helpers.GetGleanerBucketObjects(minioHandle.Client, "summoned/")
	require.NoError(t, err)
	// the third summon should not add any new files
	require.Equal(t, len(summoned1), len(summoned3))

	dateChecks = true
	sizeChecks = true
	res, msg = test_helpers.SameObjects(t, sumInfo2, sumInfo3, dateChecks, sizeChecks)
	require.True(t, res, msg)
	res, msg = test_helpers.SameObjects(t, sumInfo1, sumInfo3, dateChecks, sizeChecks)
	require.True(t, res, msg)

}
