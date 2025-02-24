package shapes

import (
	"io"
	"net/http"

	log "github.com/sirupsen/logrus"
)

// ShapeRef holds http:// or file:// URIs for shape file locations
type ShapeRef struct {
	Ref string
}

// unused for now
// func rdf2rdf(r string) (string, error) {
// 	// Decode the existing triples
// 	var inFormat rdf.Format = rdf.Turtle

// 	var outFormat rdf.Format = rdf.NTriples

// 	var s string
// 	buf := bytes.NewBufferString(s)

// 	dec := rdf.NewTripleDecoder(strings.NewReader(r), inFormat)
// 	tr, err := dec.DecodeAll()
// 	if err != nil {
// 		return "", err
// 	}

// 	enc := rdf.NewTripleEncoder(buf, outFormat)
// 	err = enc.EncodeAll(tr)

// 	enc.Close()

// 	if err != nil {
// 		return "", err
// 	}

// 	return buf.String(), err
// }

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
