package summoner

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gleaner/internal/common"
	"gleaner/internal/config"

	"github.com/minio/minio-go/v7"
	log "github.com/sirupsen/logrus"

	"gleaner/internal/summoner/acquire"

	"github.com/spf13/viper"
)

// Summoner pulls the resources from the data facilities
func SummonSitemaps(mc *minio.Client, v1 *viper.Viper) error {

	start := time.Now()
	log.Info("Summoner start time:", start) // Log the time at start for the record
	runStats := common.NewRunStats()

	// Confusingly this function was originally written such that it should not return an error if there are no API sources
	// it should just skip retrieving then and go on.
	apiSources, err := config.RetrieveSourceAPIEndpoints(v1)
	if err != nil {
		return fmt.Errorf("error getting API endpoint sources: %w", err)
	} else if len(apiSources) > 0 {
		acquire.RetrieveAPIData(apiSources, mc, runStats, v1)
	} else {
		log.Warnf("no API sources found in config file %s; this is ok if you're not using API sources", v1.ConfigFileUsed())
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		runStats.StopReason = "User Interrupt or Fatal Error"
		runStats.OutputToFile()
		os.Exit(1)
	}()

	// Get a list of resource URLs that do and don't require headless processing
	domainToUrls, err := acquire.ResourceURLs(v1, mc, false)
	if err != nil {
		log.Error("Error getting urls that do not require headless processing:", err)
	}
	// just report the error, and then run gathered urls
	if len(domainToUrls) > 0 {
		acquire.ResRetrieve(v1, mc, domainToUrls, runStats) // TODO  These can be go funcs that run all at the same time..
	}

	hru, err := acquire.ResourceURLs(v1, mc, true)
	if err != nil {
		log.Info("Error getting urls that require headless processing:", err)
	}
	// just report the error, and then run gathered urls
	if len(hru) > 0 {
		log.Info("running headless:")
		acquire.HeadlessNG(v1, mc, hru, runStats)
	}

	// Time report
	et := time.Now()
	diff := et.Sub(start)
	log.Info("Summoner run time:", diff.Minutes())
	runStats.StopReason = "Complete"
	runStats.OutputToFile()
	// What do I need to the "run" prov
	// the URLs indexed  []string
	// the graph generated?  "version" the graph by the build date
	// pass ru, hru, and v1 to a run prov function.
	//	RunFeed(v1, mc, et, ru, hru)  // DEV:   hook for building feed  (best place for it?)
	return err
}
