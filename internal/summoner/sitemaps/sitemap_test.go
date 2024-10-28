package sitemaps

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGeoconnexSitemapParse(t *testing.T) {
	sitemapUrls, err := GetSitemapsFromIndex("https://geoconnex.us/sitemap.xml")
	assert.NoError(t, err)
	var emptyMaps []string
	for _, url := range sitemapUrls {
		assert.NotEmpty(t, url)

		sitemap, err := GetSitemap(url)
		assert.NoError(t, err)
		if len(sitemap.URL) == 0 {
			emptyMaps = append(emptyMaps, url)
		}
	}
	// currently there is one website down in the sitemap. Otherwise, it is empty. Everything otherwise is expected
	assert.Equal(t, []string{"https://geoconnex.us/sitemap/CUAHSI/CUAHSI_HIS_Isábena_ids__0.xml"}, emptyMaps)
}
