package acquire

import (
	"gleaner/internal/config"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/temoto/robotstxt"
)

func TestGetRobotsForDomain(t *testing.T) {
	var robots = `User-agent: *
        Disallow: /cgi-bin
        Disallow: /forms
        Disallow: /api/gi-cat
        Disallow: /rocs/archives-catalog
        Crawl-delay: 10`

	mux := http.NewServeMux()

	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, req *http.Request) {
		_, err := w.Write([]byte(robots))
		if err != nil {
			log.Error(err)
		}
	})
	// generate a test server so we can capture and inspect the request
	testServer := httptest.NewServer(mux)
	defer func() { testServer.Close() }()

	t.Run("It returns an object representing robots.txt when specified", func(t *testing.T) {
		robots, err := getRobotsForDomain(testServer.URL)
		assert.NotNil(t, robots)
		assert.Nil(t, err)
		group := robots.FindGroup(EarthCubeAgent)
		assert.Equal(t, time.Duration(10000000000), group.CrawlDelay)
	})

	t.Run("It returns nil if there is an error", func(t *testing.T) {
		robots, err := getRobotsForDomain(testServer.URL + "/bad-value")
		assert.Nil(t, robots)
		assert.NotNil(t, err)
	})
}

func TestOverrideCrawlDelayFromRobots(t *testing.T) {

	robots, err := robotstxt.FromString(`User-agent: *
        Disallow: /cgi-bin
        Disallow: /forms
        Disallow: /api/gi-cat
        Disallow: /rocs/archives-catalog
        Crawl-delay: 10`)

	assert.Nil(t, err)

	t.Run("It does nothing if there is no robots.txt", func(t *testing.T) {
		source := &config.Source{}
		err := overrideCrawlDelayFromRobots(source, 0, nil)
		assert.Error(t, err) // should error
		assert.Equal(t, int64(0), source.Delay)
	})

	t.Run("It overrides the crawl delay if it is more", func(t *testing.T) {
		group := robots.FindGroup(EarthCubeAgent)
		src := &config.Source{}
		err := overrideCrawlDelayFromRobots(src, 10000, group)
		assert.NoError(t, err)
		assert.Equal(t, int64(10000), src.Delay)
	})

	t.Run("default to the robots.txt crawl delay if it is less", func(t *testing.T) {
		group := robots.FindGroup(EarthCubeAgent)
		src := &config.Source{}
		err := overrideCrawlDelayFromRobots(src, 1, group)
		assert.NoError(t, err)
		assert.Equal(t, int64(10), src.Delay)
	})

}
