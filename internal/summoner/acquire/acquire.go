package acquire

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	config "gleaner/internal/config"

	"github.com/PuerkitoBio/goquery"
	"github.com/minio/minio-go/v7"
	"github.com/spf13/viper"
)

const EarthCubeAgent = "EarthCube_DataBot/1.0"
const JSONContentType = "application/ld+json"

// ResRetrieve iterates through every domain and spawns a go routine to get the data from all the associated URLs for each
func ResRetrieve(v1 *viper.Viper, mc *minio.Client, domainToUrls map[string][]string) {
	wg := sync.WaitGroup{}

	// Why do I pass the wg pointer?   Just make a new one
	// for each domain in getDomain and us this one here with a semaphore
	// to control the loop?
	for domain, urls := range domainToUrls {
		log.Infof("Queuing %d URLs for domain: '%s'", len(urls), domain)
		log.Info("Queuing URLs for ", domain)
		log.Info("URL Count ", len(urls))
		wg.Add(1)
		go func() {
			getDomain(v1, mc, urls, domain)
			wg.Done()
		}()
	}

	wg.Wait()
	log.Infof("Completed acquire for %d domains", len(domainToUrls))
}

// all the configuration values that are relevant for retrieving JSON-LD for a specific URL
type retrievalConfig struct {
	BucketName    string
	ThreadCount   int
	Delay         int64
	HeadlessWait  int
	AcceptContent string
	JsonProfile   string
}

func getConfig(v1 *viper.Viper, sourceName string) (retrievalConfig, error) {
	bucketName, err := config.GetBucketName(v1)
	if err != nil {
		return retrievalConfig{}, err
	}

	var mcfg config.Summoner
	mcfg, err = config.ReadSummmonerConfig(v1.Sub("summoner"))

	if err != nil {
		return retrievalConfig{}, err
	}
	// Set default thread counts and global delay
	tc := mcfg.Threads
	delay := mcfg.Delay

	if delay != 0 || tc == 0 {
		tc = 1
	}

	// look for a domain specific override crawl delay
	sources, err := config.GetSources(v1)
	if err != nil {
		return retrievalConfig{}, err
	}
	source, err := config.GetSourceByName(sources, sourceName)
	acceptContent := source.AcceptContentType
	if acceptContent == "" {
		acceptContent = JSONContentType
	}
	jsonProfile := source.JsonProfile
	hw := source.HeadlessWait
	if err != nil {
		return retrievalConfig{}, err
	}

	if source.Delay != 0 && source.Delay > delay {
		delay = source.Delay
		tc = 1
		log.Info("Crawl delay set to ", delay, " for ", sourceName)
	}
	log.Info("Thread count ", tc, " delay ", delay)

	return retrievalConfig{
		BucketName:    bucketName,
		ThreadCount:   tc,
		Delay:         delay,
		HeadlessWait:  hw,
		AcceptContent: acceptContent,
		JsonProfile:   jsonProfile,
	}, nil
}

func getDomain(v1 *viper.Viper, mc *minio.Client, urls []string, sourceName string) {

	cfg, err := getConfig(v1, sourceName)
	if err != nil {
		// trying to read a source, so let'ss not kill everything with a panic/fatal
		log.Fatal("Error reading config file ", err)
	}

	var client http.Client

	semaphoreChan := make(chan struct{}, cfg.ThreadCount) // a blocking channel to keep concurrency under control
	lwg := sync.WaitGroup{}

	defer func() {
		lwg.Wait()
		close(semaphoreChan)
	}()

	// we actually go get the URLs now
	for i := range urls {
		lwg.Add(1)
		urlloc := urls[i]

		// TODO / WARNING for large site we can exhaust memory with just the creation of the
		// go routines. 1 million =~ 4 GB  So we need to control how many routines we
		// make too..
		go func(i int, sourceName string) {
			semaphoreChan <- struct{}{}

			log.Debug("Indexing ", urlloc)

			req, err := http.NewRequest("GET", urlloc, nil)
			if err != nil {
				log.Error(i, err, urlloc)
			}
			req.Header.Set("User-Agent", EarthCubeAgent)
			req.Header.Set("Accept", cfg.AcceptContent)

			resp, err := client.Do(req)
			if err != nil {
				log.Error("#", i, " error on ", urlloc, err) // print an message containing the index (won't keep order)
				lwg.Done()                                   // tell the wait group that we be done
				<-semaphoreChan
				return
			}
			defer resp.Body.Close()

			log.Tracef("Got statuscode %d when fetching URL: '%s'", resp.StatusCode, urlloc)
			jsonlds, err := FindJSONInResponse(v1, urlloc, cfg.JsonProfile, resp)
			// there was an issue with sitemaps... but now this code
			//if contains(contentTypeHeader, JSONContentType) || contains(contentTypeHeader, "application/json") {
			//
			//	b, err := io.ReadAll(resp.Body)
			//	// b, err := ioutil.ReadAll(resp.Body)  Go.1.15 and earlier
			//	if err != nil {
			//		log.Error("#", i, " error on ", urlloc, err) // print an message containing the index (won't keep order)
			//		repoStats.Inc(common.Issues)
			//		lwg.Done() // tell the wait group that we be done
			//		<-semaphoreChan
			//		return
			//	}
			//	jsonlds = []string{string(b)}
			//} else {
			//	var err error
			//	jsonlds, err = FindJSONInResponse(v1, urlloc, jsonProfile, repologger, resp)
			//	if err != nil {
			//		log.Error("#", i, " error on ", urlloc, err) // print an message containing the index (won't keep order)
			//		repoStats.Inc(common.Issues)
			//		lwg.Done() // tell the wait group that wes be done
			//		<-semaphoreChan
			//		return
			//	}
			//}
			if err != nil {
				log.Error("#", i, " error on ", urlloc, err) // print an message containing the index (won't keep order)
				lwg.Done()                                   // tell the wait group that we be done
				<-semaphoreChan
				return
			}

			// For incremental indexing I want to know every URL I visit regardless
			// if there is a valid JSON-LD document or not.   For "full" indexing we
			// visit ALL URLs.  However, many will not have JSON-LD, so let's also record
			// and avoid those during incremental calls.

			// even is no JSON-LD packages found, record the event of checking this URL
			if len(jsonlds) < 1 {
				// TODO is her where I then try headless, and scope the following for into an else?
				if cfg.HeadlessWait >= 0 {
					log.WithFields(log.Fields{"url": urlloc, "contentType": "Direct access failed, trying headless']"}).Info("Direct access failed, trying headless for ", urlloc)
					err := PageRenderAndUpload(v1, mc, 60*time.Second, urlloc, sourceName) // TODO make delay configurable
					if err != nil {
						log.WithFields(log.Fields{"url": urlloc, "issue": "converting json ld"}).Error("PageRenderAndUpload ", urlloc, "::", err)
					}
				}

			} else {
				log.WithFields(log.Fields{"url": urlloc, "issue": "Direct access worked"}).Trace("Direct access worked for ", urlloc)
			}

			UploadWithLogsAndMetadata(v1, mc, cfg.BucketName, sourceName, urlloc, jsonlds)

			time.Sleep(time.Duration(cfg.Delay) * time.Millisecond) // sleep a bit if directed to by the provider

			lwg.Done()

			<-semaphoreChan // clear a spot in the semaphore channel for the next indexing event
		}(i, sourceName)

	}
}

func FindJSONInResponse(v1 *viper.Viper, urlloc string, jsonProfile string, response *http.Response) ([]string, error) {
	// NewDocumentResponse is deprecated but the alternative doesn't seem to work
	body := response.Body
	if body == nil {
		return nil, fmt.Errorf("body not found on response")
	}

	doc, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		return nil, err
	}

	contentTypeHeader := response.Header["Content-Type"]
	var jsonlds []string

	// if the URL is sending back JSON-LD correctly as application/ld+json
	// this should not be here IMHO, but need to support people not setting proper header value
	// The URL is sending back JSON-LD but incorrectly sending as application/json
	// would like to add contains(contentTypeHeader, jsonProfile)
	// but empty profile strings matching all
	if contains(contentTypeHeader, JSONContentType) || contains(contentTypeHeader, "application/json") || fileExtensionIsJson(urlloc) {
		logFields := log.Fields{"url": urlloc, "contentType": "json or ld_json"}
		log.WithFields(logFields).Debug(urlloc, " as ", contentTypeHeader)
		resp_text := doc.Text()
		jsonlds, err = addToJsonListIfValid(v1, jsonlds, resp_text)
		if err != nil {
			log.WithFields(logFields).Error("Error processing json response from ", urlloc, err)
		}
		// look in the HTML response for <script type=application/ld+json> ^
	} else {
		//doc.Find("script[type='application/ld+json']").Each(func(i int, s *goquery.Selection) {
		//Please note that Cascadia's selectors do not necessarily match all supported selectors of jQuery (Sizzle).  https://github.com/andybalholm/cascadia
		doc.Find("script[type^='application/ld+json']").Each(func(i int, s *goquery.Selection) {
			jsonlds, err = addToJsonListIfValid(v1, jsonlds, s.Text())
			logFields := log.Fields{"url": urlloc, "contentType": "script[type='application/ld+json']"}
			if err != nil {
				log.WithFields(logFields).Error("Error processing script tag in ", urlloc, err)
			}
		})
	}

	return jsonlds, nil
}

// Wrap the minio PutObject function with verbose logging and track the stats
func UploadWithLogsAndMetadata(v1 *viper.Viper, mc *minio.Client, bucketName string, sourceName string, urlloc string, jsonlds []string) {

	for i, jsonld := range jsonlds {
		if jsonld == "" {
			logFields := log.Fields{"url": urlloc, "issue": "Empty JSON-LD document found "}
			log.WithFields(logFields).Info("Empty JSON-LD document found. Continuing.")
			continue
		}

		logFields := log.Fields{"url": urlloc, "issue": "Uploading"}
		log.WithFields(logFields).Trace("#", i, "Uploading ")
		sha, err := Upload(v1, mc, bucketName, sourceName, urlloc, jsonld)

		if err != nil {
			logFields = log.Fields{"url": urlloc, "sha": sha, "issue": "Error uploading jsonld to object store"}
			log.WithFields(logFields).Error("Error uploading jsonld to object store: ", urlloc, err)
		} else {
			logFields = log.Fields{"url": urlloc, "sha": sha, "issue": "Uploaded to object store"}
			log.WithFields(logFields).Trace("Successfully put ", sha, " in summoned bucket for ", urlloc)
		}
	}
}

func contains(arr []string, str string) bool {
	for _, a := range arr {

		if strings.Contains(a, str) {
			return true
		}
	}
	return false
}

func fileExtensionIsJson(rawUrl string) bool {
	u, _ := url.Parse(rawUrl)
	if strings.HasSuffix(u.Path, ".json") || strings.HasSuffix(u.Path, ".jsonld") {
		return true
	}
	return false
}
