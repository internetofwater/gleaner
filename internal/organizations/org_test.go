package organizations

import (
	"testing"

	config "gleaner/cmd/config"
	"gleaner/internal"
	"gleaner/testHelpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

func TestBuildJSONLDFromSource(t *testing.T) {

	source := config.SourceConfig{
		Name: "test",
		Url:  "https://test.com/test.xml",
	}

	jsonld, err := BuildOrgJSONLD(source)
	require.NoError(t, err)
	assert.NotEmpty(t, jsonld)
	assert.Contains(t, jsonld, source.Name)
	assert.Contains(t, jsonld, source.URL)
	assert.Contains(t, jsonld, source.PID)

	// make sure that a source without a name | url | pid results in an error
	invalidSource := config.Source{}

	_, err = BuildOrgJSONLD(invalidSource)
	assert.Error(t, err)
}

// makes a graph from the Gleaner config file source
// and upload it to minio as n-quads
func TestOrgNQsInMinio(t *testing.T) {

	conf, err := config.ReadGleanerConfig("justMainstems.yaml", "../../testHelpers/sampleConfigs")
	require.NoError(t, err)

	minioHelper, err := testHelpers.NewMinioHandle("minio/minio:latest")
	require.NoError(t, err)

	err = internal.MakeBuckets(minioHelper.Client, "gleanerbucket")
	require.NoError(t, err)

	defer func() {
		err = testcontainers.TerminateContainer(minioHelper.Container)
		assert.NoError(t, err)
	}()

	err = SummonOrgs(minioHelper.Client, conf)
	require.NoError(t, err)

	require.NoError(t, err)
	testHelpers.AssertObjectCount(t, minioHelper.Client, "orgs/", len(conf.Sources))

}
