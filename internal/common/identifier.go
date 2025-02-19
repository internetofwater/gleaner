package common

/* info on possible packages:
https://cburgmer.github.io/json-path-comparison/
using https://github.com/ohler55/ojg

test your jsonpaths here:
http://jsonpath.herokuapp.com/
There are four implementations... so you can see if one might be a little quirky
*/
import (
	"errors"
	"fmt"
	"gleaner/internal/config"
	"sort"
	"strings"

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

	// Generate calls also do the casecading aka if IdentifierSha is [] it calls JsonSha
	switch source.IdentifierType {
	case config.IdentifierString:
		return GenerateIdentiferString(v1, source, jsonld)
	case config.IdentifierSha:
		return GenerateIdentifierSha(v1, source, jsonld)
	case config.NormalizedJsonSha:
		return GenerateNormalizedSha(v1, jsonld)
	default: //config.filesha
		return GenerateFileSha(v1, jsonld)

	}

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

func GenerateIdentiferString(v1 *viper.Viper, source config.Source, jsonld string) (Identifier, error) {
	uniqueid, err := GenerateIdentifierSha(v1, source, jsonld)

	if err != nil {
		return uniqueid, err
	}
	if uniqueid.MatchedString != "" {
		uniqueid.UniqueId = uniqueid.MatchedString
		uniqueid.IdentifierType = config.IdentifierString

	}
	return uniqueid, err
}

func GenerateIdentifierSha(v1 *viper.Viper, source config.Source, jsonld string) (Identifier, error) {
	// need a copy of the arrays, or it will get munged in a multithreaded env
	var jsonpath = make([]string, len(jsonPathsDefault))
	copy(jsonpath, jsonPathsDefault)

	if len(source.IdentifierPath) > 0 && source.IdentifierPath != "" {
		// this does not move an item to the front of the array, if the item already exists in the array,
		// overriding the default overrides all paths
		//jsonpath = append(source.IdentifierPath, jsonPathsDefault...)
		//jsonpath = source.IdentifierPath
		paths := strings.Split(source.IdentifierPath, ",")
		for _, p := range paths {
			jsonpath = config.MoveToFront(p, jsonpath)
		}

	}
	jsonsha, err := GenerateNormalizedSha(v1, jsonld)
	if err != nil {
		return jsonsha, err
	}
	uniqueid, foundPath, err := GetIdentiferByPaths(jsonpath, jsonld)

	if err == nil && uniqueid != "[]" {
		id := Identifier{UniqueId: GetSHA(fmt.Sprint(uniqueid)),
			IdentifierType: config.IdentifierSha,
			MatchedPath:    foundPath,
			MatchedString:  fmt.Sprint(uniqueid),
			JsonSha:        jsonsha.JsonSha,
		}
		return id, err
	} else {
		log.Info(config.IdentifierSha, "Action: Getting normalized sha  Error:", err)
		// generate a filesha
		return GenerateNormalizedSha(v1, jsonld)
	}
}
func GenerateNormalizedSha(v1 *viper.Viper, jsonld string) (Identifier, error) {
	var id Identifier
	//uuid := common.GetSHA(jsonld)
	uuid, err := GetNormSHA(jsonld, v1) // Moved to the normalized sha value

	if uuid == "" {
		// error
		log.Error("ERROR: uuid generator:", "Action: Getting normalized sha  Error:", err)
		id = Identifier{}
	} else if err != nil {
		// no error, then normalized triples generated
		log.Info(" Action: Normalize sha generated sha:", uuid, " Error:", err)
		id = Identifier{UniqueId: uuid,
			IdentifierType: config.NormalizedJsonSha,
			JsonSha:        uuid,
		}
		err = nil
	} else {
		log.Debug(" Action: Json sha generated", uuid)
		id = Identifier{UniqueId: uuid,
			IdentifierType: config.JsonSha,
			JsonSha:        uuid,
		}
	}

	return id, err
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
