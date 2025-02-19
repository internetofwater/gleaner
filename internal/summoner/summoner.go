package summoner

import (
	"time"

	"gleaner/cmd/config"

	"github.com/minio/minio-go/v7"
	log "github.com/sirupsen/logrus"

	"gleaner/internal/summoner/acquire"
)

// Summoner pulls the resources from the data facilities
func SummonSitemaps(mc *minio.Client, conf config.GleanerConfig) error {

	start := time.Now()

	// Get a list of resource URLs that do and don't require headless processing
	domainToUrls, err := acquire.ResourceURLs(conf, mc, false)
	if err != nil {
		log.Error("Error getting urls that do not require headless processing:", err)
		return err
	}
	if len(domainToUrls) > 0 {
		acquire.ResRetrieve(conf, mc, domainToUrls)
	}

	headlessResourceURLs, err := acquire.ResourceURLs(conf, mc, true)
	if err != nil {
		log.Error("Error getting urls that require headless processing:", err)
		return err
	}
	// just report the error, and then run gathered urls
	if len(headlessResourceURLs) > 0 {
		log.Info("running headless:")
		if err := acquire.HeadlessNG(conf, mc, headlessResourceURLs); err != nil {
			return err
		}
	}

	diff := time.Now().Sub(start)
	log.Info("Summoner run time:", diff.Minutes())
	return err
}
