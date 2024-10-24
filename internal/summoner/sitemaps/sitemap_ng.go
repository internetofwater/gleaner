package sitemaps

import (
	"encoding/xml"
	"strings"

	log "github.com/sirupsen/logrus"

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

// Sitemap is a structure of <sitemap>
type Sitemap struct {
	XMLName xml.Name `xml:":urlset"`
	URL     []URL    `xml:":url"`
}

// URL is a structure of <url> in <sitemap>
type URL struct {
	Loc        string  `xml:"loc"`
	LastMod    string  `xml:"lastmod"`
	ChangeFreq string  `xml:"changefreq"`
	Priority   float32 `xml:"priority"`
}

func DomainSitemap(sm string) (Sitemap, error) {
	// result := make([]string, 0)
	smsm := Sitemap{}

	urls := make([]URL, 0)
	err := sitemap.ParseFromSite(sm, func(entry sitemap.Entry) error {
		url := URL{}
		url.Loc = strings.TrimSpace(entry.GetLocation())
		//TODO why is this failing?  The string doesn't exist..  need to test and trap
		// 	entry.LastMod = e.GetLastModified().String()
		// entry.ChangeFreq = strings.TrimSpace(e.GetChangeFrequency())
		urls = append(urls, url)
		return nil
	})

	if err != nil {
		log.Error(err)
	}

	smsm.URL = urls
	return smsm, err
}

// Get the domain from the sitemap
func DomainIndex(sitemapURL string) ([]string, error) {
	result := make([]string, 0)
	err := sitemap.ParseIndexFromSite(sitemapURL, func(e sitemap.IndexEntry) error {
		result = append(result, strings.TrimSpace(e.GetLocation()))
		return nil
	})

	return result, err
}
