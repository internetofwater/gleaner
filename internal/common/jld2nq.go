package common

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/piprate/json-gold/ld"
)

// JLD2nq converts JSON-LD documents to NQuads
func JLD2nq(jsonld string, proc *ld.JsonLdProcessor, options *ld.JsonLdOptions) (string, error) {
	var arbitraryJSONLD interface{}
	err := json.Unmarshal([]byte(jsonld), &arbitraryJSONLD)
	if err != nil {
		log.Error(err)
		return "", err
	}

	nQuads, err := proc.ToRDF(arbitraryJSONLD, options)
	if err != nil {
		log.Error(err)
		return "", err
	}

	switch nQuads := nQuads.(type) {
	case string:
		return nQuads, err
	default:
		return "", fmt.Errorf("nq is not a string, instead it is a %T with value %v", nQuads, nQuads)
	}
}
