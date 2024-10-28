package acquire

import (
	"errors"
	"gleaner/internal/common"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	configTypes "gleaner/internal/config"

	"github.com/temoto/robotstxt"

	"gleaner/internal/summoner/sitemaps"

	"github.com/minio/minio-go/v7"
	"github.com/spf13/viper"
)

// Sources Holds the metadata associated with the sites to harvest
//
//	type Sources struct {
//		Name       string
//		Logo       string
//		URL        string
//		Headless   bool
//		PID        string
//		ProperName string
//		Domain     string
//		// SitemapFormat string
//		// Active        bool
//	}
//
// type Sources = configTypes.Sources

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
			log.Error("Mode diff is not currently supported")
			//urls = excludeAlreadySummoned(domain.Name, urls)
		}
		overrideCrawlDelayFromRobots(v1, domain.Name, mcfg.Delay, group)
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
		overrideCrawlDelayFromRobots(v1, domain.Name, mcfg.Delay, group)
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
		log.Info(domainURL, " is not a sitemap index, checking to see if it is a sitemap")
		us, err = sitemaps.GetSitemap(domainURL)
		if err != nil {
			log.Error("Error parsing sitemap index at ", domainURL, err)
			return s, err
		}
	} else {
		log.Info("Walking the sitemap index for sitemaps")
		for _, idxv := range idxr {
			subset, err := sitemaps.GetSitemap(idxv)
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
		return errors.New("no robots.txt for " + sourceName)
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
