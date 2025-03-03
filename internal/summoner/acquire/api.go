package acquire

import (
	configTypes "gleaner/internal/config"
	"sync"

	"github.com/minio/minio-go/v7"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// given a paged API url template, concurrently iterate through the pages until we get
// all the results we want.
func RetrieveAPIData(apiSources []configTypes.Source, mc *minio.Client, v1 *viper.Viper) {
	wg := sync.WaitGroup{}

	for _, source := range apiSources {
		log.Info("Queuing API calls for ", source.Name)
		panic("not implemented")
	}

	wg.Wait()
}
