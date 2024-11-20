package millers

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"gleaner/internal/config"
	"gleaner/internal/millers/graph"

	"github.com/minio/minio-go/v7"
	"github.com/spf13/viper"
)

type Sources struct {
	Name     string
	Logo     string
	URL      string
	Headless bool
	// SitemapFormat string
	// Active        bool
}

// Millers is our main controller for calling the various milling paths we will
// do on the JSON-LD data graphs
func Millers(mc *minio.Client, v1 *viper.Viper) {
	st := time.Now()
	log.Info("Miller start time", st) // Log the time at start for the record

	// Put the sources in the config file into a struct
	//var domains []Sources
	//err := v1.UnmarshalKey("sources", &domains)

	//domains, err := config.GetSources(v1)
	domains, err := config.GetActiveSources(v1)
	if err != nil {
		log.Error(err)
	}

	activeBuckets := []string{}
	for i := range domains {
		m := fmt.Sprintf("summoned/%s", domains[i].Name)
		activeBuckets = append(activeBuckets, m)
		log.Info("Adding bucket to milling list:", m)
	}

	// Make array of prov buckets..  sad I have to do this..  I could just pass
	// the domains and let each miller now pick where to get things from.  I
	// only had to add this due to the prov data not being in summoned

	provBuckets := []string{}
	for i := range domains {
		m := fmt.Sprintf("prov/%s", domains[i].Name)
		provBuckets = append(provBuckets, m)
		log.Info("Adding bucket to prov building list:", m)
	}

	mcfg := v1.GetStringMapString("millers") // get the millers we want to run from the config file

	// Graph is the miller to convert from JSON-LD to nquads with validation of well formed
	// TODO  none of these (graph, shacl, prov) deal with the returned error
	if mcfg["graph"] == "true" {
		for d := range activeBuckets {
			err := graph.GraphNG(mc, activeBuckets[d], v1)
			if err != nil {
				log.Error(err)
			}
		}
	}

	// if mcfg["shacl"] == "true" {
	// 	for d := range as {
	// 		shapes.ShapeNG(mc, as[d], v1)
	// 		// shapes.SHACLMillObjects(mc, as[d], v1)
	// 	}
	// }

	// if mcfg["prov"] == "true" {
	// 	for d := range ap {
	// 		graph.AssembleObjs(mc, ap[d], v1)
	// 	}
	// }

	// Time report
	et := time.Now()
	diff := et.Sub(st)
	log.Info("Miller end time:", et)
	log.Info("Miller run time:", diff.Minutes())
}
