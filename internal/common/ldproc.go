package common

// this needs to have some wrapper around normalize to throw an error when things are not right

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"

	"gleaner/internal/projectpath"

	"github.com/piprate/json-gold/ld"
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
func NewJSONLDProcessor(contextMapping []ContextMapping, cache bool) (*ld.JsonLdProcessor, *ld.JsonLdOptions, error) {
	proc := ld.NewJsonLdProcessor()
	options := ld.NewJsonLdOptions("")

	// if the user wants to cache the context, use a caching document loader
	if cache {
		client := &http.Client{}
		nl := ld.NewDefaultDocumentLoader(client)

		m := make(map[string]string)

		for i := range contextMapping {
			if fileExistsRelativeToRoot(contextMapping[i].File) {
				// make the path absolute to the Root so we
				// don't need to deal with relative paths
				// affecting behavior different in prod vs testing
				contextMapping[i].File = filepath.Join(projectpath.Root, contextMapping[i].File)
				m[contextMapping[i].Prefix] = contextMapping[i].File
			} else {
				return nil, nil, errors.New("context file location " + contextMapping[i].File + " does not exist")
			}
		}

		// Read mapping from config file
		cdl := ld.NewCachingDocumentLoader(nl)
		err := cdl.PreloadWithMapping(m)
		if err != nil {
			return nil, nil, err
		}
		options.DocumentLoader = cdl
	}

	options.Format = "application/nquads"

	return proc, options, nil
}

func fileExistsRelativeToRoot(filename string) bool {
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
