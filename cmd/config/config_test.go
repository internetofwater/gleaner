package config

import (
	"gleaner/internal/projectpath"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigParse(t *testing.T) {
	conf := filepath.Join(projectpath.Root, "cmd", "testdata")

	_, err := ReadGleanerConfig(conf, "gleanerConfig.yaml")
	require.NoError(t, err)
}
