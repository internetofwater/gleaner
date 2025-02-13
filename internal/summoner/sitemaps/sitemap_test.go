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

		sitemap, err := ParseSitemap(url)
		assert.NoError(t, err)
		if len(sitemap.URL) == 0 {
			emptyMaps = append(emptyMaps, url)
		}
	}
	// the array of empty sitemap names should be empty, signifying there are no empty sitemaps
	assert.Empty(t, emptyMaps)

}
