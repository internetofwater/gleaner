package config

import (
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func SetupHelper(conf map[string]interface{}) *viper.Viper {
	var viper = viper.New()
	for key, value := range conf {
		viper.Set(key, value)
	}
	return viper
}

// Read the config and get API endpoint template strings
func RetrieveSourceAPIEndpoints(v1 *viper.Viper) ([]Sources, error) {

	// Get our API sources
	sources, err := GetSources(v1)
	if err != nil {
		log.Error("Error getting sources to summon: ", err)
		return []Sources{}, err
	}

	return FilterSourcesByType(sources, "api"), nil

}

func fileNameWithoutExtTrimSuffix(fileName string) string {
	return strings.TrimSuffix(fileName, filepath.Ext(fileName))
}

// moveToFront moves needle to the front of haystack, in place if possible.
// from https://github.com/golang/go/wiki/SliceTricks
func MoveToFront(needle string, haystack []string) []string {
	if len(haystack) != 0 && haystack[0] == needle {
		return haystack
	}
	prev := needle
	for i, elem := range haystack {
		switch {
		case i == 0:
			haystack[0] = needle
			prev = elem
		case elem == needle:
			haystack[i] = prev
			return haystack
		default:
			haystack[i] = prev
			prev = elem
		}
	}
	return append(haystack, prev)
}
