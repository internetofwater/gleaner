package common

// this needs to have some wrapper around normalize to throw an error when things are not right

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"

	"gleaner/internal/projectpath"

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
func GenerateJSONLDProcessor(v1 *viper.Viper) (*ld.JsonLdProcessor, *ld.JsonLdOptions, error) {
	proc := ld.NewJsonLdProcessor()
	options := ld.NewJsonLdOptions("")

	contextConfig := v1.GetStringMapString("context")

	// if the user wants to cache the context, use a caching document loader
	if contextConfig["cache"] == "true" {
		client := &http.Client{}
		nl := ld.NewDefaultDocumentLoader(client)

		var contexts []ContextMapping
		err := v1.UnmarshalKey("contextmaps", &contexts)
		if err != nil {
			return nil, nil, err
		}

		m := make(map[string]string)

		for i := range contexts {
			if FileExistsRelativeToRoot(contexts[i].File) {
				// make the path absolute to the Root so we 
				// don't need to deal with relative paths
				// affecting behavior different in prod vs testing
				contexts[i].File = filepath.Join(projectpath.Root, contexts[i].File)
				m[contexts[i].Prefix] = contexts[i].File
			} else {
				return nil, nil, errors.New("context file location " + contexts[i].File + " does not exist")
			}
		}

		// Read mapping from config file
		cdl := ld.NewCachingDocumentLoader(nl)
		err = cdl.PreloadWithMapping(m)
		if err != nil {
			return nil, nil, err
		}
		options.DocumentLoader = cdl
	}

	options.Format = "application/nquads"

	return proc, options, nil
}

func FileExistsRelativeToRoot(filename string) bool {
	filename = filepath.Join(projectpath.Root, filename)

	info, err := os.Stat(filename)
	if err != nil {
		return false
	}
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
