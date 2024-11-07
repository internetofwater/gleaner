package acquire

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	log "github.com/sirupsen/logrus"

	configTypes "gleaner/internal/config"

	"gleaner/internal/common"
	"gleaner/internal/millers/graph"

	"github.com/minio/minio-go/v7"
	"github.com/spf13/viper"
)

// Downloads pre-built site graphs
func LoadSiteSitegraphsIfExist(mc *minio.Client, v1 *viper.Viper) (string, error) {

	bucketName, err := configTypes.GetBucketName(v1)
	if err != nil {
		return "", err
	}

	sources, err := configTypes.GetSources(v1)
	if err != nil {
		log.Error(err)
	}
	domainsToCrawl := configTypes.FilterSourcesByType(sources, "sitegraph")
	if len(domainsToCrawl) == 0 {
		return "", fmt.Errorf("no sitegraph sources found")
	}

	for _, domain := range domainsToCrawl {
		log.Info("Processing sitegraph file (this can be slow with little feedback):", domain.URL)
		log.Info("Downloading sitegraph file:", domain.URL)

		d, err := getJSON(domain.URL)
		if err != nil {
			fmt.Println("error with reading graph JSON: " + domain.URL)
		}

		// TODO, how do we quickly validate the JSON-LD files to make sure it is at least formatted well

		sha := common.GetSHA(d) // Don't normalize big files..

		// Upload the file
		log.Info("Sitegraph file downloaded. Uploading to", bucketName, domain.URL)

		objectName := fmt.Sprintf("summoned/%s/%s.jsonld", domain.Name, sha)
		_, err = graph.LoadToMinio(d, bucketName, objectName, mc)
		if err != nil {
			return objectName, err
		}
		log.Info("Sitegraph file uploaded to", bucketName, "Uploaded :", domain.URL)
		// mill the json-ld to nq and upload to minio
		// we bypass graph.GraphNG which does a time consuming blank node fix which is not required
		// when dealing with a single large file.
		// log.Print("Milling graph")
		//graph.GraphNG(mc, fmt.Sprintf("summoned/%s/", domains[k].Name), v1)
		proc, options, err := common.GenerateJSONLDProcessor(v1) // Make a common proc and options to share with the upcoming go funcs
		if err != nil {
			return "", err
		}
		rdf, err := common.JLD2nq(d, proc, options)
		if err != nil {
			return "", err
		}

		log.Info("Processed Sitegraph being uploaded to", bucketName, domain.URL)
		milledName := fmt.Sprintf("milled/%s/%s.rdf", domain.Name, sha)
		_, err = graph.LoadToMinio(rdf, bucketName, milledName, mc)
		if err != nil {
			return objectName, err
		}
		log.Info("Processed Sitegraph Upload to", bucketName, "complete:", domain.URL)

		// build prov
		if err := StoreProvNamedGraph(v1, mc, domain.Name, sha, domain.URL, "summoned"); err != nil {
			return objectName, err
		}

		log.Info("Loaded:", len(d))
	}

	return "Sitegraph(s) processed", err
}

func getJSON(urlloc string) (string, error) {

	urlloc = strings.TrimSpace(urlloc)
	//resp, err := http.Get(url)
	//if err != nil {
	//	return "", fmt.Errorf("GET error: %v", err)
	//}
	/*  https://oih.aquadocs.org/aquadocs.json  fialing with a 403.
	// this is to http 1.1 spec: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Host
	*/

	var client http.Client // why do I make this here..  can I use 1 client?  move up in the loop
	req, err := http.NewRequest("GET", urlloc, nil)
	if err != nil {
		log.Error(err)
	}
	req.Header.Set("User-Agent", "EarthCube_DataBot/1.0")
	u, err := url.Parse(urlloc)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Host", u.Hostname())
	resp, err := client.Do(req)
	if err != nil {
		log.Error(" error on", urlloc, err) // print an message containing the index (won't keep order)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status error: %v", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read body: %v", err)
	}

	return string(data), nil
}
