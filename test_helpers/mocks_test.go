package test_helpers

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServeStaticFile(t *testing.T) {

	server, listener, err := ServeSampleConfigDir()
	assert.NoError(t, err)
	defer func() {
		server.Close()
		listener.Close()
	}()
	resp, err := http.Get("http://" + listener.Addr().String() + "/small_sitemap.xml")
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}
