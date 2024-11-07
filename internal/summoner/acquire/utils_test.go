package acquire

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetRobotsTxt(t *testing.T) {
	var robotsHeader = `User-agent: *
        Disallow: /cgi-bin
        Disallow: /forms
        Disallow: /api/gi-cat
        Disallow: /rocs/archives-catalog
        Crawl-delay: 10`

	mux := http.NewServeMux()

	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte(robotsHeader))
	})

	// generate a test server so we can capture and inspect the request
	testServer := httptest.NewServer(mux)
	defer func() { testServer.Close() }()

	t.Run("It returns an object representing robots.txt when specified", func(t *testing.T) {
		robots, err := getRobotsTxt(testServer.URL + "/robots.txt")
		assert.NotNil(t, robots)
		assert.Nil(t, err)
		assert.False(t, robots.TestAgent("/cgi-bin/exploit", "*"))
	})

	t.Run("It returns nil if there is no robots.txt at that url", func(t *testing.T) {
		robots, err := getRobotsTxt(testServer.URL + "/404.txt")
		assert.Nil(t, robots)
		assert.Error(t, err)
	})
}
