package acquire

import (
	"context"
	"net/http"
	"testing"
	"time"

	"fmt"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

type HeadlessContainer struct {
	mappedPort int
	url        string
	Container  *testcontainers.Container
}

// Spin up a local graphdb container and the associated client
func NewHeadlessContainer() (HeadlessContainer, error) {
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "chromedp/headless-shell:latest",
		ExposedPorts: []string{"9222/tcp"},
		Name:         "gleanerHeadlessTestcontainer",
	}
	graphdbC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
		Reuse:            true,
	})

	if err != nil {
		return HeadlessContainer{}, err
	}
	// 9222 is the default http endpoint
	port, err := graphdbC.MappedPort(ctx, "9222/tcp")

	if err != nil {
		return HeadlessContainer{}, err
	}

	return HeadlessContainer{mappedPort: port.Int(), url: "http://localhost:" + fmt.Sprint(port.Int()), Container: &graphdbC}, nil
}

func (c *HeadlessContainer) Ping() (int, error) {

	var client = http.Client{
		Timeout: 2 * time.Second,
	}

	req, err := http.NewRequest("HEAD", c.url, nil)
	if err != nil {
		return 0, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	resp.Body.Close()
	return resp.StatusCode, nil
}

func TestHeadlessNG(t *testing.T) {
	container, err := NewHeadlessContainer()
	require.NoError(t, err)

	status, err := container.Ping()
	t.Log(status)
	require.NoError(t, err)
	require.Equal(t, 200, status)

	tests := []struct {
		name         string
		url          string
		jsonldcount  int
		headlessWait int
		expectedFail bool
	}{
		{name: "r2r_wait_5_works_returns_2_jsonld",
			url:          "https://dev.rvdata.us/search/fileset/100135",
			jsonldcount:  2,
			headlessWait: 5,
		},
		{name: "r2r_expectedfail_wait_0_returns_1_jsonld_fails_if_2_jsonld",
			url:          "https://dev.rvdata.us/search/fileset/100135",
			jsonldcount:  2,
			headlessWait: 0,
			expectedFail: true,
		},
	}

	for _, test := range tests {

		conf := map[string]interface{}{
			"minio":    map[string]interface{}{"bucket": "test"},
			"summoner": map[string]interface{}{"threads": "5", "delay": 10, "headless": container.url},
			"sources":  []map[string]interface{}{{"name": test.name, "headlessWait": test.headlessWait}},
		}

		var viper = viper.New()
		for key, value := range conf {
			viper.Set(key, value)
		}
		t.Run(test.name, func(t *testing.T) {
			jsonlds, err := PageRender(viper, 5*time.Second, test.url, test.name)
			if !test.expectedFail {
				assert.Equal(t, test.jsonldcount, len(jsonlds))
			} else {
				assert.NotEqual(t, test.jsonldcount, len(jsonlds))
			}

			assert.Nil(t, err)

		})
	}
}
