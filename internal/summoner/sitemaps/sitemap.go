package sitemaps

import (
	"encoding/xml"
	"strings"

	sitemap "github.com/oxffaa/gopher-parse-sitemap"
)

// Index is a structure of <sitemapindex>
type Index struct {
	XMLName xml.Name `xml:"sitemapindex"`
	Sitemap []parts  `xml:"sitemap"`
}

// parts is a structure of <sitemap> in <sitemapindex>
type parts struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod"`
}

// Represents an XML sitemap
type Sitemap struct {
	XMLName xml.Name `xml:":urlset"`
	URL     []URL    `xml:":url"`
}

// Represents a URL tag and its attributes within a sitemap
type URL struct {
	Loc        string  `xml:"loc"`
	LastMod    string  `xml:"lastmod"`
	ChangeFreq string  `xml:"changefreq"`
	Priority   float32 `xml:"priority"`
}

func ParseSitemap(sitemapURL string) (Sitemap, error) {
	// result := make([]string, 0)
	serializedSitemap := Sitemap{}

	urls := make([]URL, 0)
	err := sitemap.ParseFromSite(sitemapURL, func(entry sitemap.Entry) error {
		url := URL{}
		url.Loc = strings.TrimSpace(entry.GetLocation())
		urls = append(urls, url)
		return nil
	})

	if err != nil {
		return serializedSitemap, err
	}

	serializedSitemap.URL = urls
	return serializedSitemap, nil
}

// // This function takes a top level sitemap index like geoconnex.us/sitemap.xml and returns a list of sitemap urls
// A sitemap index is a file that lists the URLs for multiple sitemaps
func GetSitemapsFromIndex(sitemapURL string) ([]string, error) {
	result := make([]string, 0)
	err := sitemap.ParseIndexFromSite(sitemapURL, func(e sitemap.IndexEntry) error {
		result = append(result, strings.TrimSpace(e.GetLocation()))
		return nil
	})

	return result, err
}
