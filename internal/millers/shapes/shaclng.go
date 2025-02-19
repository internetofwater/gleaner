package shapes

import (
	"context"
	"fmt"
	"strings"
	"sync"

	configTypes "gleaner/internal/config"

	log "github.com/sirupsen/logrus"

	"gleaner/internal/common"

	minio "github.com/minio/minio-go/v7"
	"github.com/spf13/viper"
)

// ShapeNG is a new and improved RDF conversion
func ShapeNG(mc *minio.Client, prefix string, v1 *viper.Viper) error {

	// read config file
	//miniocfg := v1.GetStringMapString("minio")
	//bucketName := miniocfg["bucket"] //   get the top level bucket for all of gleaner operations from config file
	bucketName, err := configTypes.GetBucketName(v1)
	if err != nil {
		log.Error(err)
		return err
	}

	err = loadShapeFiles(mc, v1) // TODO, this should be done in main
	if err != nil {
		log.Error(err)
		return err
	}

	// My go func controller vars
	semaphoreChan := make(chan struct{}, 30) // a blocking channel to keep concurrency under control (1 == single thread)
	defer close(semaphoreChan)
	wg := sync.WaitGroup{} // a wait group enables the main process a wait for goroutines to finish

	// params for list objects calls
	doneCh := make(chan struct{}) // , N) Create a done channel to control 'ListObjectsV2' go routine.
	defer close(doneCh)           // Indicate to our routine to exit cleanly upon return.
	isRecursive := true

	x := 0 // ugh..  why won't len(oc) work..   buffered channel issue I assume?
	opts := minio.ListObjectsOptions{
		Recursive: isRecursive,
		Prefix:    prefix,
	}
	//for range mc.ListObjectsV2(bucketname, prefix, isRecursive, doneCh)
	for range mc.ListObjects(context.Background(), bucketName, opts) {
		x = x + 1
	}

	// TODO get the list of shape files in the shape bucket
	//for shape := range mc.ListObjectsV2(bucketname, "shapes", isRecursive, doneCh) {
	opts2 := minio.ListObjectsOptions{
		Recursive: isRecursive,
		Prefix:    "shapes",
	}
	for shape := range mc.ListObjects(context.Background(), bucketName, opts2) {

		//for object := range mc.ListObjectsV2(bucketname, prefix, isRecursive, doneCh) {
		for object := range mc.ListObjects(context.Background(), prefix, minio.ListObjectsOptions{
			Recursive: isRecursive,
		}) {
			wg.Add(1)
			go func(object minio.ObjectInfo) {
				semaphoreChan <- struct{}{}
				//status := shaclTest(e[k].Urlval, e[k].Jld, m[j].Key, m[j].Jld, &gb)
				_, err := shaclTestNG(v1, bucketName, "verified", mc, object, shape)
				if err != nil {
					log.Error(err)
				}

				// _, err := obj2RDF(bucketName, prefix, mc, object, proc, options)

				wg.Done() // tell the wait group that we be done
				log.Debug("Doc:", bucketName, "error:", err)

				<-semaphoreChan
			}(object)
		}
	}
	wg.Wait()

	// uiprogress.Stop()

	// // all done..  write the full graph to the object store
	// log.Printf("Saving full graph to  gleaner milled:  Ref: %s/%s", bucketName, prefix)

	// //pipeCopyNG(mcfg["runid"], "gleaner-milled", fmt.Sprintf("%s-sg", prefix), mc)
	// // TODO fix this with correct variables
	// pipeCopyNG(mcfg["runid"], "gleaner-milled", fmt.Sprintf("%s-sg", prefix), mc)
	// log.Printf("Saving datagraph to:  %s/%s", bucketName, prefix)

	// log.Printf("Processed prefix: %s", prefix)
	millprefix := strings.ReplaceAll(prefix, "summoned", "verified")
	sp := strings.SplitAfterN(prefix, "/", 2)
	mcfg := v1.GetStringMapString("gleaner")
	rslt := fmt.Sprintf("results/%s/%s_verified.nq", mcfg["runid"], sp[1])
	log.Info("Assembling result graph for prefix:", prefix, "to:", millprefix)
	log.Info("Result graph will be at:", rslt)

	err = common.PipeCopyNamedGraph(rslt, bucketName, millprefix, mc)
	if err != nil {
		log.Error("Error on pipe copy:", err)
	} else {
		log.Info("Pipe copy for shacl done")
	}

	return err
}
