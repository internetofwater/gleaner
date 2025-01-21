package testHelpers

import (
	"fmt"
	"gleaner/internal/config"
	"gleaner/internal/projectpath"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServeStaticFile(t *testing.T) {

	server, listener, err := ServeSampleConfigDir()
	assert.NoError(t, err)
	defer func() {
		server.Close()
		listener.Close()
	}()

	resp, err := http.Get("http://" + listener.Addr().String() + "/sitemapIndex.xml")
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	abbreviatedSitemap := "mainstemSitemapWithoutMost.xml"

	sitemapUrl := fmt.Sprintf("http://%s/%s", listener.Addr().String(), abbreviatedSitemap)

	resp, err = http.Get(sitemapUrl)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

// Make sure that we can set the url dynamically in a tmp config so
// that we can serve it in our mocked endpoints and not need to hard code
// it and risk conflicting ports
func TestMockConfig(t *testing.T) {

	base := filepath.Join(projectpath.Root, "testHelpers", "sampleConfigs")
	confName, err := NewTempConfig("justMainstemsLocalEndpoint.yml", base)
	require.NoError(t, err)
	defer os.Remove(confName)

	err = MutateYamlSourceUrl(confName, 0, "http://example.com")
	require.NoError(t, err)

	// split the config path to get the filename
	// get the basename of the config file
	base = filepath.Base(confName)
	dir := filepath.Dir(confName)

	v, err := config.ReadGleanerConfig(base, dir)
	require.NoError(t, err)

	require.Equal(t, "http://example.com", v.GetString("sources.0.url"))

}
