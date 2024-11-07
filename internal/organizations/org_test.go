package organizations

import (
	"testing"

	"gleaner/internal/check"
	config "gleaner/internal/config"
	"gleaner/test_helpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

func TestBuildJSONLDFromSource(t *testing.T) {

	source := config.Source{
		Name: "test",
		URL:  "https://test.com/test.xml",
		PID:  "https://test.com",
	}

	jsonld, err := BuildOrgJSONLD(source)
	require.NoError(t, err)
	assert.NotEmpty(t, jsonld)

	// make sure that a source without a name | url | pid results in an error
	invalidSource := config.Source{}

	_, err = BuildOrgJSONLD(invalidSource)
	assert.Error(t, err)
}

// makes a graph from the Gleaner config file source
// and upload it to minio as n-quads
func TestOrgNQsInMinio(t *testing.T) {

	// read config file
	v1, err := config.ReadGleanerConfig("just_mainstems.yaml", "../../test_helpers/sample_configs")
	assert.NoError(t, err)

	minioHelper, err := test_helpers.NewMinioHandle("minio/minio:latest")
	require.NoError(t, err)

	err = check.MakeBuckets(minioHelper.Client, "gleanerbucket")
	require.NoError(t, err)

	defer testcontainers.TerminateContainer(minioHelper.Container)

	err = BuildOrgNqsAndUpload(minioHelper.Client, v1)
	assert.NoError(t, err)

	sources, err := config.GetSources(v1)
	assert.NoError(t, err)
	test_helpers.AssertObjectCount(t, minioHelper.Client, "orgs/", len(sources))

}
