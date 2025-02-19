package acquire

import (
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	configTypes "gleaner/cmd/config"

	"github.com/temoto/robotstxt"

	"gleaner/internal/summoner/sitemaps"

	"github.com/minio/minio-go/v7"
	"github.com/spf13/viper"
)

type SourceWithRobots struct {
	Source configTypes.SourceConfig
	Robots *robotstxt.RobotsData
}

// Gets the resource URLs for a domain.  The results is a
// map with domain name as key and []string of the URLs to process.
func ResourceURLs(conf configTypes.GleanerConfig, mc *minio.Client, headless bool) (map[string][]string, error) {
	domainsMap := make(map[string][]string)

	for _, domain := range conf.GetHeadlessSources() {
		var robots *robotstxt.RobotsData
		var group *robotstxt.Group

		if conf.Rude == true {
			robots = nil
			group = nil
			log.Info("Rude indexing mode enabled; ignoring robots.txt.")
		} else {
			robots, err := getRobotsForDomain(domain.Domain)
			if err != nil {
				log.Error("Error getting robots.txt for ", domain.Name, ", continuing without it.")
				robots = nil
				group = nil
			}
		}
		if robots != nil {
			group = robots.FindGroup(EarthCubeAgent)
			log.Info("Got robots.txt group ", group)
		}
		urls, err := getSitemapURLList(domain.Url, group)
		if err != nil {
			log.Error("Error getting sitemap urls for: ", domain.Name, err)
			return nil, err
		}
		if conf.Mode == "diff" {
			return nil, fmt.Errorf("Mode diff is not currently supported")
		}
		err = overrideCrawlDelayFromRobots(v1, domain.Name, mcfg.Delay, group)
		if err != nil {
			return nil, err
		}
		domainsMap[domain.Name] = urls
		log.Debug(domain.Name, "sitemap size is :", len(domainsMap[domain.Name]), " mode: ", mcfg.Mode)
	}
	return domainsMap, nil
}

// given a sitemap url, parse it and get the list of URLS from it.
func getSitemapURLList(domainURL string, robots *robotstxt.Group) ([]string, error) {
	var us sitemaps.Sitemap
	var s []string

	idxr, err := sitemaps.GetSitemapsFromIndex(domainURL)
	if err != nil {
		log.Error("Error reading sitemap at:", domainURL, err)
		return s, err
	}

	if len(idxr) < 1 {
		us, err = sitemaps.ParseSitemap(domainURL)
		if err != nil {
			log.Error(domainURL, " could not be parsed as either a sitemap index or a sitemap", err)
			return s, err
		}
		log.Info(domainURL, " was parsed as a sitemap")

	} else {
		log.Info("Walking the sitemap index for sitemaps")
		for _, idxv := range idxr {
			subset, err := sitemaps.ParseSitemap(idxv)
			us.URL = append(us.URL, subset.URL...)
			if err != nil {
				log.Error("Error parsing sitemap index at: ", idxv, err)
				return s, err
			}
		}
	}

	// Convert the array of sitemap package struct to simply the URLs in []string
	for _, urlStruct := range us.URL {
		if urlStruct.Loc != "" { // TODO why did this otherwise add a nil to the array..  need to check
			loc := strings.TrimSpace(urlStruct.Loc)
			loc = strings.ReplaceAll(loc, " ", "")
			loc = strings.ReplaceAll(loc, "\n", "")

			if robots != nil && !robots.Test(loc) {
				log.Error("Declining to index ", loc, " because it is disallowed by robots.txt. Error information, if any:", err)
				continue
			}
			s = append(s, loc)
		}
	}

	return s, nil
}

func overrideCrawlDelayFromRobots(v1 *viper.Viper, sourceName string, delay int64, robots *robotstxt.Group) error {
	if robots == nil {
		log.Warnf("No robots.txt found for %s so no crawl delay will be set", sourceName)
		return nil
	}

	// Look at the crawl delay from this domain's robots.txt, if we can, and one exists.
	// this is a time.Duration, which is in nanoseconds but we want milliseconds
	log.Debug("Raw crawl delay for robots ", sourceName, " set to ", robots.CrawlDelay)
	crawlDelay := int64(robots.CrawlDelay / time.Millisecond)
	log.Debug("Crawl Delay specified by robots.txt for ", sourceName, " : ", crawlDelay)

	// If our default delay is less than what is set there, set a delay for this
	// domain to respect the robots.txt setting.
	if delay < crawlDelay {
		sources, err := configTypes.GetSources(v1)
		if err != nil {
			log.Fatal(err)
		}
		source, err := configTypes.GetSourceByName(sources, sourceName)

		if err != nil {
			return err
		}
		source.Delay = crawlDelay
		v1.Set("sources", sources)
	}
	return nil
}

func getRobotsForDomain(url string) (*robotstxt.RobotsData, error) {
	robotsUrl := url + "/robots.txt"
	log.Info("Getting robots.txt from ", robotsUrl)
	robots, err := getRobotsTxt(robotsUrl)
	if err != nil {
		log.Info("error getting robots.txt for ", url, ":", err)
		return nil, err
	}
	return robots, nil
}
