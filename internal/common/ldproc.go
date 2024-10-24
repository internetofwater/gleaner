package common

// this needs to have some wrapper around normalize to throw an error when things are not right

import (
	"errors"
	"net/http"
	"os"

	"github.com/piprate/json-gold/ld"
	"github.com/spf13/viper"
)

// ContextMapping holds the JSON-LD mappings for cached context
type ContextMapping struct {
	Prefix string
	File   string
}

// JLDProc builds the JSON-LD processer and sets the options object
// to use in framing, processing and all JSON-LD actions
// TODO   we create this all the time..  stupidly..  Generate these pointers
// and pass them around, don't keep making it over and over
// Ref:  https://schema.org/docs/howwework.html and https://schema.org/docs/jsonldcontext.json
func JLDProc(v1 *viper.Viper) (*ld.JsonLdProcessor, *ld.JsonLdOptions, error) {
	proc := ld.NewJsonLdProcessor()
	options := ld.NewJsonLdOptions("")

	mcfg := v1.GetStringMapString("context")

	// if the user wants to cache the context, use a caching document loader
	if mcfg["cache"] == "true" {
		client := &http.Client{}
		nl := ld.NewDefaultDocumentLoader(client)

		var s []ContextMapping
		err := v1.UnmarshalKey("contextmaps", &s)
		if err != nil {
			return nil, nil, err
		}

		m := make(map[string]string)

		for i := range s {
			if fileExists(s[i].File) {
				m[s[i].Prefix] = s[i].File

			} else {
				return nil, nil, errors.New("context file location " + s[i].File + " does not exist")
			}
		}

		// Read mapping from config file
		cdl := ld.NewCachingDocumentLoader(nl)
		cdl.PreloadWithMapping(m)
		options.DocumentLoader = cdl
	}

	options.Format = "application/nquads"

	return proc, options, nil
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if err != nil {
		return false
	}
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
