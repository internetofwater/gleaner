package common

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

/*
This is to test various identifier
It uses a structure of expectations to run a series of individual tests with the name: testname_jsonfilename.

In the future, the JSON should probably be loaded from a file in resources_test folder.
*/

/* info on possible packages:
https://cburgmer.github.io/json-path-comparison/
using https://github.com/ohler55/ojg

test your jsonpaths here:
http://jsonpath.herokuapp.com/
There are four implementations... so you can see if one might be a little quirky
*/

// testdata is in internal/common/testdata/identifier
// thoughts are that these many be migrated to  an Approval Test approach.
// gets rid of the extpectations part, and would match the entire returned identifier object.

// should record a table of the file sha and normalize triple sha for each file

type jsonexpectations struct {
	name            string
	json            map[string]string
	IdentifierType  string
	IdentifierPaths string
	expected        string
	expectedPath    string
	errorExpected   bool
	ignore          bool
}

// using idenfiters as a stand in for array of identifiers.

func testValidJsonPath(tests []jsonexpectations, t *testing.T) {
	for _, test := range tests {
		for i, json := range test.json {
			t.Run(fmt.Sprint(test.name, "_", i), func(t *testing.T) {
				if test.ignore {
					return
				}
				path := filepath.Join("testdata", "identifier", json)
				assert.FileExistsf(t, path, "Datafile Missing: {path}")
				source, err := os.ReadFile(path)
				if err != nil {
					t.Fatal("error reading source file:", err)
				}

				result, err := GetIdentifierByPath(test.IdentifierPaths, string(source))
				valStr := fmt.Sprint(result)
				assert.Equal(t, test.expected, valStr)
				assert.Nil(t, err)
			})
		}
	}

	//t.Run("@id", func(t *testing.T) {
	//
	//	result, err := GetIdentifierByPath(sources[0].IdentifierPath, jsonId)
	//	valStr := fmt.Sprint(result)
	//	assert.Equal(t, "[idenfitier]", valStr)
	//	assert.Nil(t, err)
	//})
	//t.Run(".idenfitier", func(t *testing.T) {
	//
	//	result, err := GetIdentifierByPath(sources[1].IdentifierPath, jsonId)
	//	valStr := fmt.Sprint(result)
	//	assert.Equal(t, "[doi:10.1575/1912/bco-dmo.2343.1]", valStr)
	//	assert.Nil(t, err)
	//})
	//t.Run("$.idenfitier", func(t *testing.T) {
	//
	//	result, err := GetIdentifierByPath(sources[2].IdentifierPath, jsonId)
	//	valStr := fmt.Sprint(result)
	//	assert.Equal(t, "[doi:10.1575/1912/bco-dmo.2343.1]", valStr)
	//	assert.Nil(t, err)
	//})
	// to do: test for valid JSON but invalid RDF triples
}

// test the array paths
func testValidJsonPaths(tests []jsonexpectations, t *testing.T) {
	for _, test := range tests {
		for i, json := range test.json {
			t.Run(fmt.Sprint(test.name, "_", i), func(t *testing.T) {
				if test.ignore {
					return
				}
				path := filepath.Join("testdata", "identifier", json)
				assert.FileExistsf(t, path, "Datafile Missing: {path}")
				source, err := os.ReadFile(path)
				if err != nil {
					t.Fatal("error reading source file:", err)
				}
				paths := strings.Split(test.IdentifierPaths, ",")
				result, foundPath, err := GetIdentiferByPaths(paths, string(source))
				valStr := fmt.Sprint(result)
				assert.Equal(t, test.expected, valStr, "expected Failed")
				assert.Equal(t, test.expectedPath, foundPath, "matched Path Failed")
				assert.Nil(t, err)
			})
		}

	}

}

/*
this tests a single path against a single json file
*/
func TestValidJsonPathInput(t *testing.T) {

	var tests = []jsonexpectations{
		// default
		{
			name:          "@id",
			json:          map[string]string{"jsonID": "jsonId.json"},
			errorExpected: false,

			IdentifierPaths: `$['@id']`,
			expected:        "[idenfitier]",
			expectedPath:    "$['@id']",
			ignore:          false,
		},
		//https://raw.githubusercontent.com/earthcube/GeoCODES-Metadata/main/metadata/Dataset/actualdata/earthchem2.json
		{
			name:            "@.identifier",
			json:            map[string]string{"jsonID": "jsonId.json"},
			errorExpected:   false,
			IdentifierPaths: "@.identifier",
			expected:        "[doi:10.1575/1912/bco-dmo.2343.1]",
			expectedPath:    "@.identifier",
			ignore:          false,
		},
		//https://raw.githubusercontent.com/earthcube/GeoCODES-Metadata/main/metadata/Dataset/actualdata/earthchem2.json
		{
			name:            "$.identifier",
			json:            map[string]string{"jsonID": "jsonId.json"},
			errorExpected:   false,
			IdentifierPaths: "$.identifier",
			expected:        "[doi:10.1575/1912/bco-dmo.2343.1]",
			expectedPath:    "$.identifier",
			ignore:          false,
		},
		// argo example: https://raw.githubusercontent.com/earthcube/GeoCODES-Metadata/main/metadata/Dataset/actualdata/argo.json
		{
			name:            "identifiers Array ",
			json:            map[string]string{"jsonID": "jsonId.json"},
			errorExpected:   false,
			IdentifierPaths: "$.identifierSArray[?(@.propertyID=='https://registry.identifiers.org/registry/doi')].value",
			expected:        "[doi:10.1575/1912/bco-dmo.2343.1 doi:10.1575/1912/bco-dmo.2343.1N]",
			expectedPath:    "$.identifierSArray[?(@.propertyID=='https://registry.identifiers.org/registry/doi')].value",
			ignore:          false,
		},
		{
			name:          "identifier_obj",
			json:          map[string]string{"jsonID": "jsonId.json"},
			errorExpected: false,
			//	IdentifierPath: "$.identifierObj[?(@.propertyID=='https://registry.identifiers.org/registry/doi')].value",
			//IdentifierPath: "$.identifierObj.propertyID[@=='https://registry.identifiers.org/registry/doi')]",
			IdentifierPaths: "$.identifierObj.value",
			expected:        "[doi:10.1575/1912/bco-dmo.2343.1]",
			expectedPath:    "$.identifierObj.value",
			ignore:          false,
		},
		//https://raw.githubusercontent.com/earthcube/GeoCODES-Metadata/main/metadata/Dataset/actualdata/earthchem2.json
		// this will not work since the || does not work
		{
			name:            " identifier or id",
			json:            map[string]string{"jsonID": "jsonId.json"},
			errorExpected:   false,
			IdentifierPaths: "[ $.identifiers[?(@.propertyID=='https://registry.identifiers.org/registry/doi')].value || $.['@id'] ]",
			expected:        "[doi:10.1575/1912/bco-dmo.2343.1]",
			expectedPath:    "[ $.identifiers[?(@.propertyID=='https://registry.identifiers.org/registry/doi')].value || $.['@id'] ]",
			ignore:          true,
		},
		// identifier as array: https://github.com/earthcube/GeoCODES-Metadata/blob/main/metadata/Dataset/allgood/bcodmo1.json
		/* needs work
		"identifier": [

		       {
		           "@type": "PropertyValue",
		           "@id": "https://doi.org/10.1575/1912/bco-dmo.2343.1",
		           "propertyID": "https://registry.identifiers.org/registry/doi",
		           "value": "doi:10.1575/1912/bco-dmo.2343.1",
		           "url": "https://doi.org/10.1575/1912/bco-dmo.2343.1"
		       }
		   ],
		*/
		// this does not work fancy array index issues. Would be nice
		{
			name:          "identifierSArray slice",
			json:          map[string]string{"jsonID": "jsonId.json"},
			errorExpected: false,
			//IdentifierPath: "$.identifierSArray[?(@.propertyID=='https://registry.identifiers.org/registry/doi')].value[-1:]",
			//IdentifierPaths: []string{"$.identifierSArray[?(@.propertyID=='https://registry.identifiers.org/registry/doi')].value.[-1:]"},
			IdentifierPaths: "$.identifierSArray[?(@.propertyID=='https://registry.identifiers.org/registry/doi')].value[0]",
			expected:        "[doi:10.1575/1912/bco-dmo.2343.1]",
			expectedPath:    "$.identifierSArray[?(@.propertyID=='https://registry.identifiers.org/registry/doi')].value.[0]",
			ignore:          true,
		},
	}

	testValidJsonPath(tests, t)
}

func TestValidJsonPathsInput(t *testing.T) {

	var tests = []jsonexpectations{
		// default
		// should work for all
		{
			name: "@id",
			json: map[string]string{"jsonID": "jsonIdPaths.json", "jsonIdentifier": "jsonIdentifierPath.json",
				"jsonobjectId":                "jsonIdentifierObjectPath.json",
				"jsonIdentifierArraySingle":   "jsonIdentifierArraySingle.json",
				"jsonIdentifierArrayMultiple": "jsonIdentifierArrayMultiple.json",
			},
			errorExpected: false,

			IdentifierPaths: `$['@id']`,
			expected:        "[idenfitier]",
			expectedPath:    "$['@id']",
			ignore:          false,
		},
		//https://raw.githubusercontent.com/earthcube/GeoCODES-Metadata/main/metadata/Dataset/actualdata/earthchem2.json
		// this returns an empty set [] https://cburgmer.github.io/json-path-comparison/results/dot_notation_on_object_without_key.html
		{
			name: "$.identifier.$id",
			//json:            []string{jsonId},
			json: map[string]string{"jsonID": "jsonIdPaths.json"}, //"jsonIdentifier": jsonIdentifier,
			//"jsonobjectId": jsonIdentifierObject,
			//"jsonIdentifierArraySingle": jsonIdentifierArraySingle,
			//"jsonIdentifierArrayMultiple": jsonIdentifierArrayMultiple,

			errorExpected:   false,
			IdentifierPaths: "$.identifier.value,$.identifier,$['@id']",
			expected:        "[idenfitier]",
			expectedPath:    "$['@id']",
			ignore:          false,
		},
		{
			name: "$.identifier.$.identifier",
			//json:            []string{jsonIdentifier},
			json:            map[string]string{"jsonIdentifier": "jsonIdentifierPath.json"},
			errorExpected:   false,
			IdentifierPaths: "$.identifier.value,$.identifier,$['@id']",
			expected:        "[doi:10]",
			expectedPath:    "$.identifier",
			ignore:          false,
		},
		//https://raw.githubusercontent.com/earthcube/GeoCODES-Metadata/main/metadata/Dataset/actualdata/earthchem2.json
		{
			name: "$.identifierObjBracket",
			//json:            []string{jsonIdentifierObject},
			json: map[string]string{
				"jsonobjectId": "jsonIdentifierObjectPath.json",
			},
			errorExpected:   false,
			IdentifierPaths: "$.identifier['value'],$.identifier,$['@id']",
			expected:        "[doi:10.1575/1912/bco-dmo.2343.1]",
			expectedPath:    "$.identifier['value']",
			ignore:          false,
		},
		{
			name: "$.identifierObjDot",
			//json:            []string{jsonIdentifierObject},
			json: map[string]string{
				"jsonobjectId": "jsonIdentifierObjectPath.json",
			},
			errorExpected:   false,
			IdentifierPaths: "$.identifier.value,$.identifier,$['@id']",
			expected:        "[doi:10.1575/1912/bco-dmo.2343.1]",
			expectedPath:    "$.identifier.value",
			ignore:          false,
		},
		{
			name: "$.identifierObjCheck",
			//json:            []string{jsonIdentifierObject},
			json: map[string]string{
				"jsonobjectId": "jsonIdentifierObjectPath.json",
			},
			errorExpected:   false,
			IdentifierPaths: "$.identifier.value,$.identifier,$['@id']",
			expected:        "[doi:10.1575/1912/bco-dmo.2343.1]",
			expectedPath:    "$.identifier.value",
			ignore:          false,
		},
		//https://raw.githubusercontent.com/earthcube/GeoCODES-Metadata/main/metadata/Dataset/actualdata/earthchem2.json
		{
			name: "@.identifierArraySimple",
			//json:            []string{jsonIdentifierArraySingle},
			json: map[string]string{
				"jsonIdentifierArraySingle": "jsonIdentifierArraySingle.json",
			},
			errorExpected:   false,
			IdentifierPaths: "$.identifier[?(@.propertyID=='https://registry.identifiers.org/registry/doi')].value,$.identifier.value,$.identifier.$['@id']",
			expected:        "[doi:10.1575/1912/bco-dmo.2343.1]",
			expectedPath:    "$.identifier[?(@.propertyID=='https://registry.identifiers.org/registry/doi')].value",
			ignore:          false,
		},

		//https://raw.githubusercontent.com/earthcube/GeoCODES-Metadata/main/metadata/Dataset/actualdata/earthchem2.json
		{
			name: "@.identifierArrayMultiple",
			//json:            []string{jsonIdentifierArrayMultiple},
			json: map[string]string{
				"jsonIdentifierArrayMultiple": "jsonIdentifierArrayMultiple.json",
			},
			errorExpected:   false,
			IdentifierPaths: "$.identifier[?(@.propertyID=='https://registry.identifiers.org/registry/doi')].value,$.identifier.value,$.identifier,$['@id']",
			expected:        "[doi:10.1575/1912/bco-dmo.2343.1 doi:10.1575/1912/bco-dmo.2343.1N]",
			expectedPath:    "$.identifier[?(@.propertyID=='https://registry.identifiers.org/registry/doi')].value",
			ignore:          false,
		},
		{
			name: "@.identifierProblemChildIris",
			//json:            []string{jsonIdentifierArrayMultiple},
			json: map[string]string{
				"problem child": "problemChildIris.json",
			},
			errorExpected:   false,
			IdentifierPaths: "$.identifier[?(@.propertyID=='https://registry.identifiers.org/registry/doi')].value,$.identifier.value,$.identifier,$['@id']",
			expected:        "[https://ds.iris.edu/ds/products/emtf/]",
			expectedPath:    "$['@id']",
			ignore:          false,
		},
		{
			name: "@.identifierProblemChildOpenTopo",
			//json:            []string{jsonIdentifierArrayMultiple},
			json: map[string]string{
				"problem child opentopo": "problemChildOpentop.json",
			},
			errorExpected:   false,
			IdentifierPaths: "$.identifier[?(@.propertyID=='https://registry.identifiers.org/registry/doi')].value,$.identifier.value,$.identifier,$['@id']",
			expected:        "[OTDS.062020.32611.1]",
			expectedPath:    "$.identifier.value",
			ignore:          false,
		},
	}
	testValidJsonPaths(tests, t)
}

func TestValidJsonPathGraphInput(t *testing.T) {

	var tests = []jsonexpectations{
		// default

		{
			name:          "identifieGraph Not Graph",
			json:          map[string]string{"jsonID": "jsonIdentifierArrayMultiple.json"},
			errorExpected: true,
			//IdentifierPath: "$.identifierSArray[?(@.propertyID=='https://registry.identifiers.org/registry/doi')].value[-1:]",
			IdentifierPaths: "$['@graph'][?(@['@type']=='schema:Dataset')]['@id'],$.identifier[?(@.propertyID=='https://registry.identifiers.org/registry/doi')].value,$.identifier.value,$.identifier,$['@id']",

			expected:     "[doi:10.1575/1912/bco-dmo.2343.1 doi:10.1575/1912/bco-dmo.2343.1N]",
			expectedPath: "$.identifier[?(@.propertyID=='https://registry.identifiers.org/registry/doi')].value",
			ignore:       false,
		},
		// grr. Ugly since the herokuapp no longer runs: used this a hint, then raw debugging: https://cburgmer.github.io/json-path-comparison/

		// this one $['@graph]*[?(@['@type']=='schema:Dataset')]  gives false here: https://jsonpath.curiousconcept.com/
		// $['@graph']*.['@type'] returns types
		// $['@graph'].*.@id returns types
		//$.@graph*[?(@.@type=="schema:Dataset")] false bad when debuggin. cannot start with an @

		// workslocally:
		// returns nil: "$['@graph']*[?(@['@type']=='schema:Dataset')]"
		// returns full object: "$['@graph'][?(@['@type']=='schema:Dataset')]"
		// returns @id: "$['@graph'][?(@['@type']=='schema:Dataset')]['@id']"  fails at: https://jsonpath.curiousconcept.com/
		{
			name:          "identifiersGraph",
			json:          map[string]string{"jsonGraph": "jsonGraphWifire.json"},
			errorExpected: false,
			//IdentifierPath: "$.identifierSArray[?(@.propertyID=='https://registry.identifiers.org/registry/doi')].value[-1:]",
			IdentifierPaths: "$['@graph'][?(@['@type']=='schema:Dataset')]['@id']",

			expected:     "[https://wifire-data.sdsc.edu/dataset/8fd44c38-f6d3-429c-a785-1498dfaa2a6a]",
			expectedPath: "$['@graph'][?(@['@type']=='schema:Dataset')]['@id']",
			ignore:       false,
		},
		{
			name:          "identifiersGraphLong",
			json:          map[string]string{"jsonGraph": "jsonGraphWifire.json"},
			errorExpected: false,
			//IdentifierPath: "$.identifierSArray[?(@.propertyID=='https://registry.identifiers.org/registry/doi')].value[-1:]",
			IdentifierPaths: "$['@graph'][?(@['@type']=='schema:Dataset')]['@id'],$.identifier[?(@.propertyID=='https://registry.identifiers.org/registry/doi')].value,$.identifier.value,$.identifier,$['@id']",

			expected:     "[https://wifire-data.sdsc.edu/dataset/8fd44c38-f6d3-429c-a785-1498dfaa2a6a]",
			expectedPath: "$['@graph'][?(@['@type']=='schema:Dataset')]['@id']",
			ignore:       false,
		},
	}

	testValidJsonPaths(tests, t)
}
