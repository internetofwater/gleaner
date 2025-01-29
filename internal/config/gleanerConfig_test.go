package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGleanerConfig(t *testing.T) {
	v, err := ReadGleanerConfig("gleanerconfig.yaml", "../../testHelpers/sampleConfigs")
	if err != nil {
		t.Fatal(err)
	}
	res := v.Sub("minio")
	if res == nil {
		t.Fatal("no minio config")
	}
	assert.Equal(t, 9000, res.GetInt("port"))
	minio, err := ReadMinioConfig(res)
	require.NoError(t, err)
	assert.Equal(t, "gleanerbucket", minio.Bucket)
}

func TestGleanerConfigInNabuRepo(t *testing.T) {
	v, err := ReadGleanerConfig("gleaner_config_in_nabu_repo.yaml", "./testdata")
	if err != nil {
		t.Fatal(err)
	}
	res := v.Sub("minio")
	if res == nil {
		t.Fatal("no minio config")
	}
	assert.Equal(t, 9000, res.GetInt("port"))

	sources, err := GetSources(v)
	require.NoError(t, err)
	if sources == nil {
		t.Fatal("no sources config")
	}
}

func TestGleanerConfigWithMinioAddress(t *testing.T) {
	v, err := ReadGleanerConfig("gleaner_config_with_minio_address.yml", "./testdata")
	if err != nil {
		t.Fatal(err)
	}
	res := v.Sub("minio")
	if res == nil {
		t.Fatal("no minio config")
	}
	assert.Equal(t, 9000, res.GetInt("port"))
	assert.Equal(t, "minio", res.GetString("address"))

	sources, err := GetSources(v)
	require.NoError(t, err)
	if sources == nil {
		t.Fatal("no sources config")
	}
}
