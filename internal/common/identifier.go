package common

/* info on possible packages:
https://cburgmer.github.io/json-path-comparison/
using https://github.com/ohler55/ojg

test your jsonpaths here:
http://jsonpath.herokuapp.com/
There are four implementations... so you can see if one might be a little quirky
*/
import (
	"crypto/sha1"
	"errors"
	"fmt"
	"gleaner/internal/config"
	"sort"

	"github.com/ohler55/ojg/jp"
	"github.com/ohler55/ojg/oj"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Identifier is the structure returned the information
type Identifier struct {
	UniqueId       string // file sha, identifier sha, or url normalized identifier
	IdentifierType string // the returned IdentifierType..
	MatchedPath    string
	MatchedString  string
	JsonSha        string
}

var jsonPathsDefault = []string{"$['@graph'][?(@['@type']=='schema:Dataset')]['@id']", "$.identifier[?(@.propertyID=='https://registry.identifiers.org/registry/doi')].value", "$.identifier.value", "$.identifier", "$['@id']", "$.url"}

func GenerateIdentifier(v1 *viper.Viper, source config.Source, jsonld string) (Identifier, error) {
	return GenerateFileSha(v1, jsonld)
}

func GetIdentifierByPath(jsonPath string, jsonld string) (interface{}, error) {
	obj, err := oj.ParseString(jsonld)
	if err != nil {
		return "", err
	}
	x, err := jp.ParseString(jsonPath)
	ys := x.Get(obj)

	if err != nil {
		return "", err
	}
	// we need to sort the results
	aString := make([]string, len(ys))
	for i, v := range ys {
		//aString[i] = v.(string)
		aString[i] = fmt.Sprint(v) // v not always a single string
	}
	sort.SliceStable(aString, func(i, j int) bool {
		return aString[i] < aString[j]
	})
	return aString, err
}

// given a set of json paths return the first to the last.
/*
Pass an array of JSONPATH, and get returned the first not empty, result
Cautions: test your paths, consensus returns [] for a $.identifer.value, even through

{ identifier:"string"}
has no value:

"idenfitier":
	{
	"@type": "PropertyValue",
	"@id": "https://doi.org/10.1575/1912/bco-dmo.2343.1",
	"propertyID": "https://registry.identifiers.org/registry/doi",
	"value": "doi:10.1575/1912/bco-dmo.2343.1",
	"url": "https://doi.org/10.1575/1912/bco-dmo.2343.1"
	}
https://cburgmer.github.io/json-path-comparison/results/dot_notation_on_object_without_key.html
https://cburgmer.github.io/json-path-comparison/results/dot_notation_on_null_value.html
*/
func GetIdentiferByPaths(jsonpaths []string, jsonld string) (interface{}, string, error) {
	for _, jsonPath := range jsonpaths {
		obj, err := GetIdentifierByPath(jsonPath, jsonld)
		if err == nil {
			// returned a string, but
			// sometimes an empty string is returned
			if fmt.Sprint(obj) == "[]" {
				continue
			}
			return obj, jsonPath, err

		} else {
			// error,
			continue
		}
	}
	return "", "", errors.New("no Match")
}

func GetSHA(s string) string {
	h := sha1.New()
	h.Write([]byte(s))
	hs := h.Sum(nil)
	return fmt.Sprintf("%x", hs)
}

func GenerateFileSha(v1 *viper.Viper, jsonld string) (Identifier, error) {
	var id Identifier
	//uuid := common.GetSHA(jsonld)
	uuid := GetSHA(jsonld) // Moved to the normalized sha value

	if uuid == "" {
		return id, errors.New("could not generate uuid as a sha")
	}
	log.Debug(" Action: Json sha generated", uuid)
	id = Identifier{UniqueId: uuid,
		IdentifierType: config.JsonSha,
		JsonSha:        uuid,
	}

	//	fmt.Println("\njsonsha:", id)
	return id, nil
}
