package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// auth fails if a region is set in minioclient...
var gleanerTemplate = map[string]interface{}{
	"minio": map[string]string{
		"address":   "",
		"port":      "",
		"region":    "",
		"accesskey": "",
		"secretkey": "",
	},
	"gleaner":     map[string]string{},
	"context":     map[string]string{},
	"contextmaps": map[string]string{},
	"summoner":    map[string]string{},
	"millers":     map[string]string{},
	"sources": map[string]string{
		"sourcetype": "sitemap",
		"name":       "",
		"url":        "",
		"logo":       "",
		"headless":   "",
		"pid":        "",
		"propername": "",
		"domain":     "",
	},
}

func ReadGleanerConfig(filename string, cfgDir string) (*viper.Viper, error) {
	v := viper.New()
	for key, value := range gleanerTemplate {
		v.SetDefault(key, value)
	}

	v.SetConfigName(fileNameWithoutExtTrimSuffix(filename))
	v.AddConfigPath(cfgDir)
	v.SetConfigType("yaml")
	err := v.ReadInConfig()
	if err != nil {
		return v, fmt.Errorf("error when parsing gleaner config: %w", err)
	}
	return v, err
}
