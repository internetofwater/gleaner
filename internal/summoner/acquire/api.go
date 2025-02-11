package acquire

import (
	"fmt"
	"gleaner/internal/common"
	configTypes "gleaner/internal/config"
	"net/http"
	"sync"
	"time"

	"github.com/minio/minio-go/v7"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// given a paged API url template, concurrently iterate through the pages until we get
// all the results we want.
func RetrieveAPIData(apiSources []configTypes.Source, mc *minio.Client, runStats *common.RunStats, v1 *viper.Viper) {
	wg := sync.WaitGroup{}

	for _, source := range apiSources {
		r := runStats.Add(source.Name)
		r.Set(common.HttpError, 0)
		r.Set(common.Issues, 0)
		r.Set(common.Summoned, 0)
		log.Info("Queuing API calls for ", source.Name)

		repologger, err := common.LogIssues(v1, source.Name)
		if err != nil {
			log.Error("Error creating a logger for a repository", err)
		} else {
			repologger.Info("Queuing API calls for ", source.Name)
		}
		wg.Add(1)
		go getAPISource(v1, mc, source, &wg, repologger, r)
	}

	wg.Wait()
}

// Download a single API source
func getAPISource(v1 *viper.Viper, mc *minio.Client, source configTypes.Source, wg *sync.WaitGroup, repologger *log.Logger, repoStats *common.RepoStats) {

	cfg, err := getConfig(v1, source.Name) // _ is headless wait
	if err != nil {
		// trying to read a source, so let's not kill everything with a panic/fatal
		log.Error("Error reading config file ", err)
		repologger.Error("Error reading config file ", err)
	}

	var client http.Client

	responseStatusChan := make(chan int, cfg.ThreadCount) // a blocking channel to keep concurrency under control
	lwg := sync.WaitGroup{}

	defer func() {
		lwg.Wait()
		wg.Done()
		close(responseStatusChan)
	}()

	// Loop through our paged API template until
	// we get an error; i is the page number in this case
	status := http.StatusOK // start off with an OK default
	i := 0
	for status == http.StatusOK && (source.ApiPageLimit == 0 || i < source.ApiPageLimit) {
		lwg.Add(1)
		urlloc := fmt.Sprintf(source.URL, i)

		go func(i int, sourceName string) {
			repologger.Trace("Indexing", urlloc)
			log.Debug("Indexing ", urlloc)
			req, err := http.NewRequest("GET", urlloc, nil)
			if err != nil {
				log.Error(i, err, urlloc)
				return
			}
			req.Header.Set("User-Agent", EarthCubeAgent)
			req.Header.Set("Accept", cfg.AcceptContent)
			response, err := client.Do(req)

			if err != nil {
				log.Error("#", i, " error on ", urlloc, err) // print an message containing the index (won't keep order)
				repologger.WithFields(log.Fields{"url": urlloc}).Error(err)
				lwg.Done()
				responseStatusChan <- http.StatusBadRequest
				return
			}

			if response.StatusCode != http.StatusOK {
				log.Error("#", i, " response status ", response.StatusCode, " from ", urlloc)
				repologger.WithFields(log.Fields{"url": urlloc}).Error(response.StatusCode)
				lwg.Done()
				responseStatusChan <- response.StatusCode
				return
			}

			defer response.Body.Close()
			log.Trace("Response status ", response.StatusCode, " from ", urlloc)
			responseStatusChan <- response.StatusCode

			jsonlds, err := FindJSONInResponse(v1, urlloc, cfg.JsonProfile, repologger, response)

			if err != nil {
				log.Error("#", i, " error on ", urlloc, err) // print an message containing the index (won't keep order)
				repoStats.Inc(common.Issues)
				lwg.Done() // tell the wait group that we be done
				responseStatusChan <- http.StatusBadRequest
				return
			}

			// even if no JSON-LD packages found, record the event of checking this URL
			if len(jsonlds) < 1 {
				log.WithFields(log.Fields{"url": urlloc, "contentType": "No JSON-LD found']"}).Info("No JSON-LD found at ", urlloc)
				repologger.WithFields(log.Fields{"url": urlloc, "contentType": "No JSON-LD found']"}).Error() // this needs to go into the issues file

			} else {
				log.WithFields(log.Fields{"url": urlloc, "issue": "Indexed"}).Trace("Indexed ", urlloc)
				repologger.WithFields(log.Fields{"url": urlloc, "issue": "Indexed"}).Trace()
				repoStats.Inc(common.Summoned)
			}

			UploadWithLogsAndMetadata(v1, mc, cfg.BucketName, sourceName, urlloc, repologger, repoStats, jsonlds)

			log.Trace("#", i, "thread for", urlloc)                 // print an message containing the index (won't keep order)
			time.Sleep(time.Duration(cfg.Delay) * time.Millisecond) // sleep a bit if directed to by the provider

			lwg.Done()
		}(i, source.Name)
		status = <-responseStatusChan
		i++
	}
	common.RunRepoStatsOutput(repoStats, source.Name)
}
