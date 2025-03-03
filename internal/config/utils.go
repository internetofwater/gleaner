package config

import (
	"path/filepath"
	"strings"

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

func fileNameWithoutExtTrimSuffix(fileName string) string {
	return strings.TrimSuffix(fileName, filepath.Ext(fileName))
}
