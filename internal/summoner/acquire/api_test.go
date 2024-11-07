package acquire

import (
	"testing"

	"gleaner/internal/config"
	configTypes "gleaner/internal/config"

	"github.com/stretchr/testify/assert"
)

func TestRetrieveSourceAPIEndpoints(t *testing.T) {
	t.Run("It reads a config for an API indexing source and returns the expected information", func(t *testing.T) {
		apiSource := configTypes.Source{
			Name:         "apiSource",
			SourceType:   "api",
			Active:       true,
			ApiPageLimit: 42,
		}
		conf := map[string]interface{}{
			"sources": []map[string]interface{}{
				{
					"name":       "testSource",
					"sourcetype": "test",
					"active":     "true",
				},
				{
					"name":       "sitemapSource",
					"sourcetype": "sitemap",
					"active":     "true",
				},
				{
					"name":         "apiSource",
					"sourcetype":   "api",
					"apipagelimit": 42,
					"active":       "true",
				},
			},
		}

		viper := config.SetupHelper(conf)
		sources, err := config.RetrieveSourceAPIEndpoints(viper)
		assert.Equal(t, []configTypes.Source{apiSource}, sources)
		assert.Nil(t, err)
	})
}
