package acquire

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	config "gleaner/cmd/config"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestGetConfig(t *testing.T) {
	t.Run("It reads a config for an indexing source and returns the expected information", func(t *testing.T) {
		conf := map[string]interface{}{
			"minio":    map[string]interface{}{"bucket": "test"},
			"summoner": map[string]interface{}{"threads": "5", "delay": 0},
			"sources":  []map[string]interface{}{{"name": "testSource"}},
		}

		viper := config.SetupHelper(conf)
		cfg, err := getConfig(viper, "testSource")
		assert.Equal(t, "test", cfg.BucketName)
		assert.Equal(t, 5, cfg.ThreadCount)
		assert.Equal(t, int64(0), cfg.Delay)
		assert.Nil(t, err)
	})

	t.Run("It sets the thread count to 1 if a delay is specified", func(t *testing.T) {
		conf := map[string]interface{}{
			"minio":    map[string]interface{}{"bucket": "test"},
			"summoner": map[string]interface{}{"threads": "5", "delay": 1000},
			"sources":  []map[string]interface{}{{"name": "testSource"}},
		}

		viper := config.SetupHelper(conf)
		cfg, err := getConfig(viper, "testSource")
		assert.Equal(t, "test", cfg.BucketName)
		assert.Equal(t, 1, cfg.ThreadCount)
		assert.Equal(t, int64(1000), cfg.Delay)
		assert.Nil(t, err)
	})

	t.Run("It allows delay to be optional", func(t *testing.T) {
		conf := map[string]interface{}{
			"minio":    map[string]interface{}{"bucket": "test"},
			"summoner": map[string]interface{}{"threads": "5"},
			"sources":  []map[string]interface{}{{"name": "testSource"}},
		}

		viper := config.SetupHelper(conf)
		cfg, err := getConfig(viper, "testSource")
		assert.Equal(t, "test", cfg.BucketName)
		assert.Equal(t, 5, cfg.ThreadCount)
		assert.Equal(t, int64(0), cfg.Delay)
		assert.Nil(t, err)
	})

	t.Run("It overrides a global summoner delay if the data source has a longer one specified", func(t *testing.T) {
		conf := map[string]interface{}{
			"minio":    map[string]interface{}{"bucket": "test"},
			"summoner": map[string]interface{}{"threads": "5", "delay": 5},
			"sources":  []map[string]interface{}{{"name": "testSource", "delay": 100}},
		}

		viper := config.SetupHelper(conf)
		cfg, err := getConfig(viper, "testSource")
		assert.Equal(t, "test", cfg.BucketName)
		assert.Equal(t, 1, cfg.ThreadCount)
		assert.Equal(t, int64(100), cfg.Delay)
		assert.Nil(t, err)
	})

	t.Run("It does not override a global summoner delay if the data source does not have a longer one specified", func(t *testing.T) {
		conf := map[string]interface{}{
			"minio":    map[string]interface{}{"bucket": "test"},
			"summoner": map[string]interface{}{"threads": "5", "delay": 50},
			"sources":  []map[string]interface{}{{"name": "testSource", "delay": 10}},
		}

		viper := config.SetupHelper(conf)
		cfg, err := getConfig(viper, "testSource")
		assert.Equal(t, "test", cfg.BucketName)
		assert.Equal(t, 1, cfg.ThreadCount)
		assert.Equal(t, int64(50), cfg.Delay)
		assert.Nil(t, err)
	})
}

func TestFindJSONInResponse(t *testing.T) {
	conf := map[string]interface{}{
		"contextmaps": map[string]interface{}{},
	}
	viper := config.SetupHelper(conf)
	logger := log.New()
	const JSONContentType = "application/ld+json"
	testJson := `{
	    "@graph":[
	        {
	            "@context": {
	                "SO":"http://schema.org/"
	            },
	            "@type":"bar",
	            "SO:name":"Some type in a graph"
	        }
	    ]
	}`

	urlloc := "http://test"
	req, _ := http.NewRequest("GET", urlloc, nil)
	response := &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Request:    req,
		Header:     make(http.Header, 0),
	}

	t.Run("It returns an error if the response document cannot be parsed", func(t *testing.T) {
		// create dummy response object
		result, err := FindJSONInResponse(viper, urlloc, JSONContentType, logger, &http.Response{})
		assert.Nil(t, result)
		assert.Error(t, err)
	})

	t.Run("It finds JSON-LD in HTML document responses", func(t *testing.T) {
		html := "<html><body>yay<script type='application/ld+json'>" + testJson + "</script></body></html>"

		response.Body = io.NopCloser(bytes.NewBufferString(html))
		response.ContentLength = int64(len(html))
		var expected []string

		result, err := FindJSONInResponse(viper, urlloc, JSONContentType, logger, response)
		assert.Nil(t, err)
		assert.Equal(t, result, append(expected, testJson))
	})

	t.Run("It finds JSON-LD in JSON document responses", func(t *testing.T) {
		response.Body = io.NopCloser(bytes.NewBufferString(testJson))
		response.ContentLength = int64(len(testJson))
		var expected []string

		result, err := FindJSONInResponse(viper, "test.json", JSONContentType, logger, response)
		assert.Nil(t, err)
		assert.Equal(t, result, append(expected, testJson))
	})

	t.Run("It finds JSON-LD in http responses with a JSON-LD content type", func(t *testing.T) {
		response.Body = io.NopCloser(bytes.NewBufferString(testJson))
		response.ContentLength = int64(len(testJson))
		response.Header.Add("Content-Type", JSONContentType)
		var expected []string

		result, err := FindJSONInResponse(viper, urlloc, JSONContentType, logger, response)
		assert.Nil(t, err)
		assert.Equal(t, result, append(expected, testJson))
	})

	t.Run("It finds JSON-LD in http responses with a JSON content type", func(t *testing.T) {
		response.Body = io.NopCloser(bytes.NewBufferString(testJson))
		response.ContentLength = int64(len(testJson))
		response.Header.Add("Content-Type", "application/json; charset=utf-8")
		var expected []string

		result, err := FindJSONInResponse(viper, urlloc, JSONContentType, logger, response)
		assert.Nil(t, err)
		assert.Equal(t, result, append(expected, testJson))
	})

}
