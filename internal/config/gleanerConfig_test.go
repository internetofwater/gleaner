package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGleanerConfig(t *testing.T) {
	v, err := ReadGleanerConfig("gleanerconfig.yaml", "../../test_helpers/sample_configs")
	if err != nil {
		t.Fatal(err)
	}
	res := v.Sub("minio")
	if res == nil {
		t.Fatal("no minio config")
	}
	assert.Equal(t, 9000, res.GetInt("port"))
}
