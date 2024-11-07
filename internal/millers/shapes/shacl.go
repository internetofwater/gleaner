package shapes

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"

	configTypes "gleaner/internal/config"

	log "github.com/sirupsen/logrus"

	"gleaner/internal/common"
	"gleaner/internal/millers/graph"

	"github.com/knakk/rdf"
	minio "github.com/minio/minio-go/v7"
	"github.com/spf13/viper"
)

// ShapeRef holds http:// or file:// URIs for shape file locations
type ShapeRef struct {
	Ref string
}

// SHACLMillObjects test a concurrent version of calling mock
func SHACLMillObjects(mc *minio.Client, bucketname string, v1 *viper.Viper) {
	// load the SHACL files listed in the config file
	loadShapeFiles(mc, v1)

	entries := common.GetMillObjects(mc, bucketname)
	multiCall(entries, bucketname, mc, v1)
}

func loadShapeFiles(mc *minio.Client, v1 *viper.Viper) error {

	// read config file
	//miniocfg := v1.GetStringMapString("minio")
	//bucketName := miniocfg["bucket"] //   get the top level bucket for all of gleaner operations from config file
	bucketName, err := configTypes.GetBucketName(v1)
	var s []ShapeRef
	err = v1.UnmarshalKey("shapefiles", &s)
	if err != nil {
		log.Error(err)
	}

	for x := range s {
		if isURL(s[x].Ref) {
			log.Trace("Load SHACL file")
			b, err := getBody(s[x].Ref)
			if err != nil {
				log.Error("Error getting SHACL file body", err)
			}

			as := strings.Split(s[x].Ref, "/")
			// TODO  caution..  we need to note the RDF encoding and perhaps pass it along or verify it
			// is what we should be using
			_, err = graph.LoadToMinio(string(b), bucketName, fmt.Sprintf("shapes/%s", as[len(as)-1]), mc)
			if err != nil {
				log.Error(err)
			}
			log.Debug("Loaded SHACL file:", s[x].Ref)
		} else { // see if it's a file
			log.Trace("Load file...")

			dat, err := ioutil.ReadFile(s[x].Ref)
			if err != nil {
				log.Error("Error loading file", s[x].Ref, err)
			}

			as := strings.Split(s[x].Ref, "/")
			_, err = graph.LoadToMinio(string(dat), bucketName, fmt.Sprintf("shapes/%s", as[len(as)-1]), mc)
			if err != nil {
				log.Error(err)
			}
			log.Debug("Loaded SHACL file:", s[x].Ref)

		}
	}

	return nil
}

func isURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}

func multiCall(e []common.Entry, bucketname string, mc *minio.Client, v1 *viper.Viper) {
	mcfg := v1.GetStringMapString("gleaner")

	semaphoreChan := make(chan struct{}, 20) // a blocking channel to keep concurrency under control (1 == single thread)
	defer close(semaphoreChan)
	wg := sync.WaitGroup{} // a wait group enables the main process a wait for goroutines to finish

	var gb common.Buffer
	m := common.GetShapeGraphs(mc, "gleaner") // TODO: beware static bucket lists, put this in the config

	for j := range m {
		log.Debug("Checking data graphs against shape graph:", m[j])
		for k := range e {
			wg.Add(1)
			log.Trace("Ready JSON-LD package  #", j, e[k].Urlval)
			go func(j, k int) {
				semaphoreChan <- struct{}{}
				status := shaclTest(e[k].Urlval, e[k].Jld, m[j].Key, m[j].Jld, &gb)

				wg.Done() // tell the wait group that we be done
				log.Debug("#", j, e[k].Urlval, "wrote", status, "bytes")

				<-semaphoreChan
			}(j, k)
		}
	}
	wg.Wait()

	// log.Println(gb.Len())

	// TODO   gb is type turtle here..   need to convert to ntriples to store
	// nt, err := rdf2rdf(gb.String())
	// if err != nil {
	// 		log.Error(err)
	// 	}

	// write to S3
	_, err := graph.LoadToMinio(gb.String(), "gleaner-milled", fmt.Sprintf("%s/%s_shacl.nt", mcfg["runid"], bucketname), mc)
	if err != nil {
		log.Error(err)
	}
}

func rdf2rdf(r string) (string, error) {
	// Decode the existing triples
	var inFormat rdf.Format
	inFormat = rdf.Turtle

	var outFormat rdf.Format
	outFormat = rdf.NTriples

	var s string
	buf := bytes.NewBufferString(s)

	dec := rdf.NewTripleDecoder(strings.NewReader(r), inFormat)
	tr, err := dec.DecodeAll()

	enc := rdf.NewTripleEncoder(buf, outFormat)
	err = enc.EncodeAll(tr)

	enc.Close()

	return buf.String(), err
}

// this same function is in pkg/summoner  resolve duplication here and
// potentially elsewhere
func getBody(url string) ([]byte, error) {
	var client http.Client
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Error(err) // not even being able to make a req instance..  might be a fatal thing?
		return nil, err
	}

	req.Header.Set("User-Agent", "EarthCube_DataBot/1.0")

	resp, err := client.Do(req)
	if err != nil {
		log.Error("Error reading sitemap:", err)
		return nil, err
	}
	defer resp.Body.Close()

	var bodyBytes []byte
	if resp.StatusCode == http.StatusOK {
		bodyBytes, err = io.ReadAll(resp.Body)
		if err != nil {
			log.Error(err)
			return nil, err
		}
	}

	return bodyBytes, err
}
