package acquire

import (
	"fmt"
	"gleaner/internal/common"
	"gleaner/internal/config"
	"math"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	configTypes "gleaner/internal/config"

	"github.com/temoto/robotstxt"

	"gleaner/internal/summoner/sitemaps"

	"github.com/minio/minio-go/v7"
	"github.com/spf13/viper"
)

// Gets the resource URLs for a domain.  The results is a
// map with domain name as key and []string of the URLs to process.
func ResourceURLs(v1 *viper.Viper, mc *minio.Client, headless bool) (map[string][]string, error) {
	domainsMap := make(map[string][]string)
	var repoFatalErrors common.MultiError
	// Know whether we are running in diff mode, in order to exclude urls that have already
	// been summoned before
	mcfg, err := configTypes.ReadSummmonerConfig(v1.Sub("summoner"))
	if err != nil {
		return nil, err
	}
	sources, err := configTypes.GetSources(v1)
	domains := configTypes.FilterSourcesByHeadless(sources, headless)
	if err != nil {
		log.Error("Error getting sources to summon: ", err)
		return domainsMap, err // if we can't read list, ok to return an error
	}

	sitemapDomains := configTypes.FilterSourcesByType(domains, "sitemap")

	for _, domain := range sitemapDomains {
		var robots *robotstxt.RobotsData
		var group *robotstxt.Group

		if v1.Get("rude") == true {
			robots = nil
			group = nil
			log.Info("Rude indexing mode enabled; ignoring robots.txt.")
		} else {
			robots, err = getRobotsForDomain(domain.Domain)
			if err != nil {
				log.Info("Error getting robots.txt for ", domain.Name, ", continuing without it.")
				robots = nil
				group = nil
			}
		}
		if robots != nil {
			group = robots.FindGroup(EarthCubeAgent)
			log.Info("Got robots.txt group ", group)
		}
		urls, err := getSitemapURLList(domain.URL, group)
		if err != nil {
			log.Error("Error getting sitemap urls for: ", domain.Name, err)
			repoFatalErrors = append(repoFatalErrors, err)
			//return domainsMap, err // returning means that domains after broken one do not get indexed.
		}
		if mcfg.Mode == "diff" {
			log.Fatal("Mode diff is not currently supported")
		}

		err = overrideCrawlDelayFromRobots(&domain, mcfg.Delay, group)
		if err != nil {
			return nil, err
		}
		domainsMap[domain.Name] = urls
		log.Debug(domain.Name, "sitemap size is :", len(domainsMap[domain.Name]), " mode: ", mcfg.Mode)
	}

	robotsDomains := configTypes.FilterSourcesByType(domains, "robots")

	for _, domain := range robotsDomains {

		var urls []string
		// first, get the robots file and parse it
		robots, err := getRobotsTxt(domain.URL)
		if err != nil {
			log.Error("Error getting sitemap location from robots.txt for: ", domain.Name, err)
			repoFatalErrors = append(repoFatalErrors, err)
			//return domainsMap, err // returning means that domains after broken one do not get indexed.
		}
		group := robots.FindGroup(EarthCubeAgent)
		log.Debug("Found user agent group ", group)
		for _, sitemap := range robots.Sitemaps {
			sitemapUrls, err := getSitemapURLList(sitemap, group)
			if err != nil {
				log.Error("Error getting sitemap urls for: ", domain.Name, err)
				repoFatalErrors = append(repoFatalErrors, err)
				//return domainsMap, err // returning means that domains after broken one do not get indexed.
			}
			urls = append(urls, sitemapUrls...)
		}
		if mcfg.Mode == "diff" {
			log.Error("Mode diff is not currently supported")
			//urls = excludeAlreadySummoned(domain.Name, urls)
		}
		err = overrideCrawlDelayFromRobots(&domain, mcfg.Delay, group)
		if err != nil {
			return nil, err
		}
		domainsMap[domain.Name] = urls
		log.Debug(domain.Name, "sitemap size from robots.txt is : ", len(domainsMap[domain.Name]), " mode: ", mcfg.Mode)
	}
	if len(repoFatalErrors) == 0 {
		return domainsMap, nil
	} else {
		return domainsMap, repoFatalErrors
	}

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

func overrideCrawlDelayFromRobots(source *config.Source, delayOverride int64, robots *robotstxt.Group) error {
	if robots == nil {
		return fmt.Errorf("no robots.txt found for %s so no crawl delay will be set", config.SourceUrl)
	}
	// Look at the crawl delay from this domain's robots.txt, if we can, and one exists.
	// this is a time.Duration, which is in nanoseconds but we want milliseconds
	log.Debug("Raw crawl delay for robots ", source.Name, " set to ", robots.CrawlDelay)
	groupDelay := int64(robots.CrawlDelay / time.Millisecond)
	log.Debug("Crawl Delay specified by robots.txt for ", source.Name, " : ", groupDelay)

	// delay is the max of the robots.txt delay or the command line delay
	source.Delay = int64(math.Max(float64(groupDelay), float64(delayOverride)))

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
