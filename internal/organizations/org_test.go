package organizations

import (
	"context"
	"testing"

	"gleaner/internal/common"
	config "gleaner/internal/config"
	"gleaner/test_helpers"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
// load this to a /sources bucket (change this to sources naming convention?)
func TestBuildGraphMem(t *testing.T) {

	// read config file
	v1, err := config.ReadGleanerConfig("gleanerconfig.yaml", "../../test_helpers/sample_configs")
	assert.NoError(t, err)

	assert.NoError(t, err)
	bucketName, err := config.GetBucketName(v1)
	assert.NoError(t, err)
	assert.Equal(t, "gleanerbucket", bucketName)

	log.Info("Building organization graph from config file")

	domains, err := config.GetSources(v1)
	assert.NoError(t, err)

	assert.Greater(t, len(domains), 0)

	_, _, err = common.GenerateJSONLDProcessor(v1)
	assert.NoError(t, err)

	ctx := context.Background()

	minioContainer, err := test_helpers.MinioRun(ctx, "minio/minio:latest")

	BuildGraph(minioContainer.Client, v1)

}
