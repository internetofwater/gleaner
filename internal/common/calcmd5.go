package common

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"gleaner/cmd/config"
	"io"
	"strings"

	"github.com/piprate/json-gold/ld"
)

// Normalize a jsonld string and return the MD5 hash
func GetNormMD5(jsonld string, conf config.GleanerConfig) (string, error) {
	proc, options, err := GenerateJSONLDProcessor(conf)
	if err != nil {
		return "", err
	}

	options.ProcessingMode = ld.JsonLd_1_1
	options.Format = "application/n-quads"
	options.Algorithm = "URDNA2015"

	// this needs to be an interface, otherwise it thinks it is a URL to get
	var myInterface interface{}
	err = json.Unmarshal([]byte(jsonld), &myInterface)
	if err != nil {
		return "", err
	}

	normalizedTriples, err := proc.Normalize(myInterface, options)
	if err != nil {
		return "", err
	}

	r := strings.NewReader(normalizedTriples.(string))

	h := md5.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}

	hs := h.Sum(nil)
	return fmt.Sprintf("%x", hs), nil
}
