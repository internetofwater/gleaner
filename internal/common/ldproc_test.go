package common

import (
	"bytes"

	"encoding/json"
	"fmt"
	"testing"

	"github.com/piprate/json-gold/ld"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileExistsRelativeToRoot(t *testing.T) {
	exists := fileExistsRelativeToRoot("./assets/schemaorg-current-https.jsonld")
	assert.True(t, exists)

	exists = fileExistsRelativeToRoot("assets/schemaorg-current-https.jsonld")
	assert.True(t, exists)

	exists = fileExistsRelativeToRoot("/assets/schemaorg-current-https.jsonld")
	assert.True(t, exists)

	exists = fileExistsRelativeToRoot("nonexist_dir/schemaorg-current-https.jsonld")
	assert.False(t, exists)
}

/*
	ldjsonprocessor.Normalize often returns "" or the same set of triples

for JSONLD Document with context or other issues.

We will need to write a wrapper around Normalize to catch these issues, and return an error.
These are tests that helped determine that Normalize was the issue.

this tests a single path against a single json file
*/
func TestNormalizeTriple(t *testing.T) {
	var jsonNoContext = `{
"@type":"bar",
"@id":"idenfitier",
"url": "http://example.com/",
"identifier": [	
	{
	"@type": "PropertyValue",
	"@id": "https://doi.org/10.1575/1912/bco-dmo.2343.1",
	"propertyID": "https://registry.identifiers.org/registry/doi",
	"value": "doi:10.1575/1912/bco-dmo.2343.1",
	"url": "https://doi.org/10.1575/1912/bco-dmo.2343.1"
	}
	
]

}`
	var jsonNoContextSimple = `
        {
            "@type":"bar",
            "SO:name":"Some type in a graph"
        }
`
	var jsonGraphFirst = `{
    "@graph":[
        {
            "@context": {
                "SO":"http://schema.org/"
            },
            "@type":"bar",
            "SO:name":"Some type in a graph"
        }
    ]
}`

	// now using approval test approach, so expected not needed
	// look in testdata for recieved and approved information
	var tests = []jsonexpectations{
		// default
		{
			name: "noContext",
			json: map[string]string{"jsonID": jsonNoContext},
			//errorExpected: true,
			errorExpected: false, // when we proxy/wrapper NormalizeTriple this, we should throw error on empty
			expected:      "",
			ignore:        false,
		},
		{
			name:          "noContextSimple",
			json:          map[string]string{"jsonID": jsonNoContextSimple},
			errorExpected: false,

			expected: "_:c14n0 <SO:name> \"Some type in a graph\" .\n_:c14n0 <http://www.w3.org/1999/02/22-rdf-syntax-ns#type> <bar> .\n",
			ignore:   false,
		},
		{
			name:          "jsonGraphFirst",
			json:          map[string]string{"jsonID": jsonGraphFirst},
			errorExpected: false,

			expected: "_:c14n0 <http://schema.org/name> \"Some type in a graph\" .\n_:c14n0 <http://www.w3.org/1999/02/22-rdf-syntax-ns#type> <bar> .\n",
			ignore:   false,
		},
	}
	testNormalizeTriple(tests, t)
}

func testNormalizeTriple(tests []jsonexpectations, t *testing.T) {
	var vipercontext = []byte(`
context:
  cache: true
contextmaps:
- file: assets/schemaorg-current-https.jsonld
  prefix: https://schema.org/
- file: assets/schemaorg-current-https.jsonld
  prefix: http://schema.org/
sources:
- sourcetype: sitemap
  
  IdentifierType: jsonsha
`)
	viperVal := viper.New()
	viperVal.SetConfigType("yaml")
	err := viperVal.ReadConfig(bytes.NewBuffer(vipercontext))
	require.NoError(t, err)

	for _, test := range tests {
		for i, jsonld := range test.json {
			t.Run(fmt.Sprint(test.name, "_", i), func(t *testing.T) {
				if test.ignore {
					return
				}

				cache := viperVal.GetStringMapString("context")["cache"] == "true"

				var contexts []ContextMapping
				err := viperVal.UnmarshalKey("contextmaps", &contexts)
				require.NoError(t, err)
				proc, options, err := NewJSONLDProcessor(contexts, cache)
				assert.NoError(t, err)

				// add the processing mode explicitly if you need JSON-LD 1.1 features
				options.ProcessingMode = ld.JsonLd_1_1
				options.Format = "application/n-quads"
				options.Algorithm = "URDNA2015"
				var myInterface interface{}
				err = json.Unmarshal([]byte(jsonld), &myInterface)
				assert.NoError(t, err)

				result, err := proc.Normalize(myInterface, options)
				assert.NoError(t, err)

				valStr := fmt.Sprint(result)
				assert.Equal(t, test.expected, valStr)
				if test.errorExpected {
					assert.NotNil(t, err)
				} else {
					assert.Nil(t, err)
				}

			})
		}
	}

}
