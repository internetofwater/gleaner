package config

import (
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Construct a viper config from a map; useful for testing
func SetupHelper(conf map[string]interface{}) *viper.Viper {
	var viper = viper.New()
	for key, value := range conf {
		viper.Set(key, value)
	}
	return viper
}

// Read the config and get API endpoint template strings
func RetrieveSourceAPIEndpoints(v1 *viper.Viper) ([]Source, error) {

	// Get our API sources
	sources, err := GetSources(v1)
	if err != nil {
		log.Error("Error getting sources to summon: ", err)
		return []Source{}, err
	}

	return FilterSourcesByType(sources, "api"), nil

}

func fileNameWithoutExtTrimSuffix(fileName string) string {
	return strings.TrimSuffix(fileName, filepath.Ext(fileName))
}
