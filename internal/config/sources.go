package config

import (
	"errors"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	IdentifierSha     string = "identifiersha"
	JsonSha           string = "jsonsha"
	NormalizedJsonSha string = "normalizedjsonsha"
	IdentifierString  string = "identifierstring"
	SourceUrl         string = "sourceurl"
)

type ContextOption int64

const (
	Strict ContextOption = iota
	Https
	Http
	//	Array
	//	Object
	StandardizedHttps
	StandardizedHttp
)
const AccceptContentType string = "application/ld+json, text/html"

func (s ContextOption) String() string {
	switch s {
	case Strict:
		return "strict"
	case Https:
		return "https"
	case Http:
		return "http"
		//	case Array:
		//		return "array"
		//	case Object:
		//		return "object"
	case StandardizedHttps:
		return "standardizedHttps"
	case StandardizedHttp:
		return "standardizedHttp"
	}
	return "unknown"
}

// as read from csv
type Source struct {
	// Valid values for SourceType: sitemap, sitegraph, csv, googledrive, api, and robots
	SourceType      string `default:"sitemap"`
	Name            string
	Logo            string
	URL             string
	Headless        bool `default:"false"`
	PID             string
	ProperName      string
	Domain          string
	Active          bool                   `default:"true"`
	CredentialsFile string                 // do not want someone's google api key exposed.
	Other           map[string]interface{} `mapstructure:",remain"`
	// SitemapFormat string
	// Active        bool

	HeadlessWait      int    // if loading is slow, wait
	Delay             int64  // A domain-specific crawl delay value
	IdentifierPath    string // JSON Path to the identifier
	ApiPageLimit      int
	IdentifierType    string
	FixContextOption  ContextOption
	AcceptContentType string `default:"application/ld+json, text/html"` // accept content type string for http request
	JsonProfile       string // jsonprofile
}

// add needed for file
type SourcesConfig struct {
	Name       string
	Logo       string
	URL        string
	Headless   bool
	PID        string
	ProperName string
	Domain     string
	// SitemapFormat string
	// Active        bool
	HeadlessWait      int    // is loading is slow, wait
	Delay             int64  // A domain-specific crawl delay value
	IdentifierPath    string // JSON Path to the identifier
	IdentifierType    string
	FixContextOption  ContextOption
	AcceptContentType string `default:"application/ld+json, text/html"` // accept content type string for http request
	JsonProfile       string // jsonprofile
}

var SourcesTemplate = map[string]interface{}{
	"sources": map[string]string{
		"sourcetype":        "sitemap",
		"name":              "",
		"url":               "",
		"logo":              "",
		"headless":          "",
		"pid":               "",
		"propername":        "",
		"domain":            "",
		"credentialsfile":   "",
		"headlesswait":      "0",
		"delay":             "0",
		"identifierpath":    "",
		"identifiertype":    JsonSha,
		"fixcontextoption":  "https",
		"acceptcontenttype": "application/ld+json, text/html",
		"jsonprofile":       "",
	},
}

// use full gleaner viper. v1.Sub("sources") fails because it is an array.
// If we need to override with env variables, then we might need to grab this patch https://github.com/spf13/viper/pull/509/files

func GetSources(g1 *viper.Viper) ([]Source, error) {
	var subtreeKey = "sources"
	var cfg []Source

	err := g1.UnmarshalKey(subtreeKey, &cfg)
	if err != nil {
		log.Fatal("error when parsing ", subtreeKey, " config: ", err)
		return nil, err
	}
	cfg = append([]Source(nil), cfg...)
	return cfg, err
}

func GetActiveSources(g1 *viper.Viper) ([]Source, error) {
	var activeSources []Source

	sources, err := GetSources(g1)
	if err != nil {
		return nil, err
	}
	for _, s := range sources {
		if s.Active {
			activeSources = append(activeSources, s)
		}
	}
	return activeSources, err
}

func GetSourceByType(sources []Source, key string) []Source {
	var sourcesSlice []Source
	for _, s := range sources {
		if s.SourceType == key {
			sourcesSlice = append(sourcesSlice, s)
		}
	}
	return sourcesSlice
}

func FilterSourcesByType(sources []Source, requestedType string) []Source {
	var sourcesSlice []Source
	for _, s := range sources {
		if s.SourceType == requestedType && s.Active {
			sourcesSlice = append(sourcesSlice, s)
		}
	}
	return sourcesSlice
}

func FilterSourcesByHeadless(sources []Source, headless bool) []Source {
	var sourcesSlice []Source
	for _, s := range sources {
		if s.Headless == headless && s.Active {
			sourcesSlice = append(sourcesSlice, s)
		}
	}
	return sourcesSlice
}

func GetSourceByName(sources []Source, name string) (*Source, error) {
	for i := 0; i < len(sources); i++ {
		if sources[i].Name == name {
			return &sources[i], nil
		}
	}
	return nil, fmt.Errorf("unable to find a source with name %s", name)
}

func SourceToNabuPrefix(sources []Source, useMilled bool) []string {
	jsonld := "summoned"
	if useMilled {
		jsonld = "milled"
	}
	var prefixes []string
	for _, s := range sources {

		switch s.SourceType {

		case "sitemap":
			prefixes = append(prefixes, fmt.Sprintf("%s/%s", jsonld, s.Name))

		case "sitegraph":
			// sitegraph not milled
			prefixes = append(prefixes, fmt.Sprintf("%s/%s", "summoned", s.Name))
		}
	}
	return prefixes
}
func SourceToNabuProv(sources []Source) []string {

	var prefixes []string
	for _, s := range sources {
		prefixes = append(prefixes, "prov/"+s.Name)
	}
	return prefixes
}

func PruneSources(v1 *viper.Viper, useSources []string) (*viper.Viper, error) {
	var finalSources []Source
	allSources, err := GetSources(v1)
	if err != nil {
		log.Fatal("error retrieving sources: ", err)
	}
	for _, s := range allSources {
		if contains(useSources, s.Name) {
			s.Active = true // we assume you want to run this, even if disabled, normally
			finalSources = append(finalSources, s)
		}
	}
	if len(finalSources) > 0 {
		v1.Set("sources", finalSources)
		return v1, err
	} else {

		return v1, errors.New("cannot find a source with the name ")
	}

}

// checks if a string is present in a slice
func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}
