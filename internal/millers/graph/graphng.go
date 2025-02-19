package graph

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	configTypes "gleaner/internal/config"
	"io"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"

	"gleaner/internal/common"

	minio "github.com/minio/minio-go/v7"
	"github.com/piprate/json-gold/ld"
	"github.com/spf13/viper"
)

// RDF conversion
func GraphNG(mc *minio.Client, prefix string, v1 *viper.Viper) error {

	bucketName, err := configTypes.GetBucketName(v1)
	if err != nil {
		return err
	}
	semaphoreChan := make(chan struct{}, 10) // a blocking channel to keep concurrency under control (1 == single thread)
	defer close(semaphoreChan)
	wg := sync.WaitGroup{} // a wait group enables the main process a wait for goroutines to finish

	proc, options, err := common.GenerateJSONLDProcessor(v1) // Make a common proc and options to share with the upcoming go funcs
	if err != nil {
		return err
	}

	// params for list objects calls
	// doneCh := make(chan struct{}) // , N) Create a done channel to control 'ListObjectsV2' go routine.
	// defer close(doneCh)           // Indicate to our routine to exit cleanly upon return.
	isRecursive := true

	objectCh := mc.ListObjects(context.Background(), bucketName, minio.ListObjectsOptions{Prefix: prefix, Recursive: isRecursive})
	// for object := range mc.ListObjects(context.Background(), bucketname, prefix, isRecursive, doneCh) {
	for object := range objectCh {
		wg.Add(1)
		go func(object minio.ObjectInfo) {
			semaphoreChan <- struct{}{}
			_, err := uploadObj2RDF(bucketName, "milled", mc, object, proc, options)
			if err != nil {
				log.Error("uploadObj2RDF", err) // need to log to an "errors" log file
			}

			wg.Done() // tell the wait group that we be done
			log.Debug("Doc:", object.Key, "error:", err)

			<-semaphoreChan
		}(object)
	}
	wg.Wait()

	// TODO make a version of PipeCopy that generates Parquet version of graph
	// TODO..  then delete milled objects?
	millprefix := strings.ReplaceAll(prefix, "summoned", "milled")
	sp := strings.SplitAfterN(prefix, "/", 2)
	mcfg := v1.GetStringMapString("gleaner")

	rslt := fmt.Sprintf("results/%s/%s_graph.nq", mcfg["runid"], sp[1])
	log.Info("Assembling result graph for prefix:", prefix, "to:", millprefix)
	log.Info("Result graph will be at:", rslt)

	err = common.PipeCopyNamedGraph(rslt, bucketName, millprefix, mc)
	if err != nil {
		log.Error("Error on pipe copy:", err)
	} else {
		log.Info("Pipe copy for graph done")
	}

	return err
}

// Gets an jsonld file from minio, converts it to RDF, and uploads it to minio
func uploadObj2RDF(bucketName, prefix string, mc *minio.Client, object minio.ObjectInfo, proc *ld.JsonLdProcessor, options *ld.JsonLdOptions) (string, error) {
	// object is an object reader
	stat, err := mc.StatObject(context.Background(), bucketName, object.Key, minio.GetObjectOptions{})
	if err != nil {
		log.Error("Error when statting", err)
		return "", err
	}
	if stat.Size > 100000 {
		log.Warn("retrieving a large object (", stat.Size, ") (this may be slow)", object.Key)
	}
	fo, err := mc.GetObject(context.Background(), bucketName, object.Key, minio.GetObjectOptions{})
	if err != nil {
		log.Error("minio.getObject", err)
		return "", err
	}

	key := object.Key // replace if new function idea works..

	var b bytes.Buffer
	bw := bufio.NewWriter(&b)

	_, err = io.Copy(bw, fo)
	if err != nil {
		log.Error("error copying:", err)
	}

	// TODO
	// Process the bytes in b to RDF (with randomized blank nodes)
	//rdf, err := common.JLD2nq(b.String(), proc, options)
	//if err != nil {
	//	return key, err
	//}
	//
	//rdfubn := GlobalUniqueBNodes(rdf)
	rdfubn, err := Obj2RDF(b.String(), proc, options)
	if err != nil {
		return key, err
	}
	milledkey := strings.ReplaceAll(key, ".jsonld", ".rdf")
	milledkey = strings.ReplaceAll(milledkey, "summoned/", "")

	// make an object with prefix like  scienceorg-dg/objectname.rdf  (where is had .jsonld before)
	objectName := fmt.Sprintf("%s/%s", prefix, milledkey)
	usermeta := make(map[string]string) // what do I want to know?
	usermeta["origfile"] = key

	// Upload the file
	_, err = LoadToMinio(rdfubn, bucketName, objectName, mc)
	if err != nil {
		return objectName, err
	}

	return objectName, nil
}

func Obj2RDF(jsonld string, proc *ld.JsonLdProcessor, options *ld.JsonLdOptions) (string, error) {

	// Process the bytes in b to RDF (with randomized blank nodes)
	rdf, err := common.JLD2nq(jsonld, proc, options)
	if err != nil {
		return "", err
	}

	rdfubn := GlobalUniqueBNodes(rdf)

	return rdfubn, nil
}
