package millershacl

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"strings"
	"sync"

	"earthcube.org/Project418/gleaner/internal/millers/millerutils"
	"earthcube.org/Project418/gleaner/internal/utils"
	minio "github.com/minio/minio-go"
)

// SHACLMillObjects test a concurrent version of calling mock
func SHACLMillObjects(mc *minio.Client, bucketname string) {
	entries := utils.GetMillObjects(mc, bucketname)
	multiCall(entries, bucketname, mc)
}

func multiCall(e []utils.Entry, bucketname string, mc *minio.Client) {
	// Set up the the semaphore and conccurancey
	semaphoreChan := make(chan struct{}, 1) // a blocking channel to keep concurrency under control (1 == single thread)
	defer close(semaphoreChan)
	wg := sync.WaitGroup{} // a wait group enables the main process a wait for goroutines to finish

	var gb utils.Buffer
	m := utils.GetMillObjects(mc, "gleaner-shacl") // todo: beware static bucket lists..

	for j := range m {
		for k := range e {
			wg.Add(1)
			log.Printf("About to run loop #%d #%d in a goroutine\n", j, k)
			go func(j, k int) {
				semaphoreChan <- struct{}{}
				status := shaclTest(e[k].Bucketname, e[k].Key, e[k].Urlval, e[k].Jld, m[j].Key, m[j].Jld, &gb)

				wg.Done() // tell the wait group that we be done
				log.Printf("#%d #%d wrote %d bytes", j, k, status)
				<-semaphoreChan
			}(j, k)
		}
	}
	wg.Wait()

	log.Println(gb.Len())

	// write to S3
	_, err := millerutils.LoadToMinio(gb.String(), "gleaner", fmt.Sprintf("%s_shacl.n3", bucketname), mc)

	// write to file
	fl, err := millerutils.WriteRDF(gb.String(), bucketname)
	if err != nil {
		log.Println("RDF file could not be written")
	} else {
		log.Printf("RDF file written len:%d\n", fl)
	}
}

func shaclTest(bucketname, key, urlval, dg, sgkey, sg string, gb *utils.Buffer) int {
	datagraph, err := millerutils.JSONLDToTTL(dg, urlval)
	if err != nil {
		log.Printf("Error in the jsonld write... %v\n", err)
		log.Printf("Nothing to do..   going home")
		return 0
	}

	// fmt.Printf("\n\n %s \n\n", datagraph)

	url := "http://localhost:7000"
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	writer.WriteField("dataref", urlval)
	writer.WriteField("shaperef", sgkey)

	part, err := writer.CreateFormFile("datag", "datagraph")
	if err != nil {
		log.Println(err)
	}
	_, err = io.Copy(part, strings.NewReader(datagraph))
	if err != nil {
		log.Println(err)
	}

	part, err = writer.CreateFormFile("shapeg", "shapegraph")
	if err != nil {
		log.Println(err)
	}
	_, err = io.Copy(part, strings.NewReader(sg))
	if err != nil {
		log.Println(err)
	}

	err = writer.Close()
	if err != nil {
		log.Println(err)
	}

	// fmt.Println("------------------------------")
	// fmt.Println(body.String())

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		log.Println(err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("User-Agent", "EarthCube_DataBot/1.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}

	// log.Println(string(b))

	// write result to buffer
	len, err := gb.Write(b)
	if err != nil {
		log.Printf("error in the buffer write... %v\n", err)
	}

	return len //  we will return the bytes count we write...
}