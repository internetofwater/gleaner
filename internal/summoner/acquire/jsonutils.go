package acquire

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"gleaner/internal/config"

	log "github.com/sirupsen/logrus"

	"gleaner/internal/common"

	minio "github.com/minio/minio-go/v7"
	"github.com/spf13/viper"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// / A utility to keep a list of JSON-LD files that we have found
// in or on a page
func addToJsonListIfValid(v1 *viper.Viper, jsonlds []string, new_json string) ([]string, error) {
	valid, err := isValid(v1, new_json)
	if err != nil {
		isValidGraphArray, jsonldsArray, _ := isGraphArray(v1, new_json)
		if isValidGraphArray {
			return append(jsonldsArray, new_json), nil
		}
		return jsonlds, fmt.Errorf("error checking for valid json: %s", err)
	}
	if !valid {

		return jsonlds, fmt.Errorf("invalid json; continuing")
	}
	return append(jsonlds, new_json), nil
}

func isGraphArray(v1 *viper.Viper, jsonld string) (bool, []string, error) {
	var errs error
	jsonlds := []string{}
	var myArray []interface{}
	err := json.Unmarshal([]byte(jsonld), &myArray)
	if err == nil {
		var myArray []map[string]interface{}
		err := json.Unmarshal([]byte(jsonld), &myArray)
		if err == nil {
			for _, j := range myArray {
				jsonld, _ := json.Marshal(j) // we just unmarshaled it.
				valid, err := isValid(v1, string(jsonld))
				if valid && err == nil {
					jsonlds = append(jsonlds, string(jsonld))
				} else {
					errs = err
				}
			}
			if len(jsonlds) > 0 {
				return true, jsonlds, errs
			}

		}
	}
	return false, jsonlds, errs
}

// Return true if the string is valid JSON-LD
func isValid(v1 *viper.Viper, jsonld string) (bool, error) {

	cache := v1.GetStringMapString("context")["cache"] == "true"

	var contexts []common.ContextMapping
	err := v1.UnmarshalKey("contextmaps", &contexts)
	if err != nil {
		return false, err
	}

	proc, options, err := common.NewJSONLDProcessor(contexts, cache)
	if err != nil {
		return false, err
	}

	var myInterface map[string]interface{}
	err = json.Unmarshal([]byte(jsonld), &myInterface)
	if err != nil {
		return false, fmt.Errorf("error in unmarshaling json: %s", err)
	}

	_, err = proc.ToRDF(myInterface, options) // returns triples but toss them, just validating
	if err != nil {                           // it's wasted cycles.. but if just doing a summon, needs to be done here
		return false, fmt.Errorf("error in JSON-LD to RDF call: %s", err)
	}

	return true, nil
}

// let's try to do them all, in one, since that will make the code a bit cleaner and easier to test
// don' think this is currently called anywhere
const httpContext = "http://schema.org/"
const httpsContext = "https://schema.org/"

// this is unused
// func fixContext(jsonld string, option config.ContextOption) (string, error) {
// 	var err error

// 	if option == config.Strict {
// 		return jsonld, nil
// 	}
// 	jsonContext := gjson.Get(jsonld, "@context")

// 	var ctxSchemaOrg = httpsContext
// 	if option == config.Http {
// 		ctxSchemaOrg = httpContext
// 	}

// 	switch reflect.ValueOf(jsonContext).Kind() {
// 	case reflect.String:
// 		jsonld, err = fixContextString(jsonld, config.Https)
// 	case reflect.Slice:
// 		jsonld, err = fixContextArray(jsonld, config.Https)
// 	}
// 	jsonld, err = fixContextUrl(jsonld, ctxSchemaOrg)
// 	return jsonld, err
// }

// Our first json fixup in existence.
// If the top-level JSON-LD context is a string instead of an object,
// this function corrects it.
func fixContextString(jsonld string) (string, error) {
	var err error
	jsonContext := gjson.Get(jsonld, "@context")

	switch jsonContext.Value().(type) {
	case string:
		jsonld, err = sjson.Set(jsonld, "@context", map[string]interface{}{"@vocab": jsonContext.String()})
	}
	return jsonld, err
}

// If the top-level JSON-LD context does not end with a trailing slash or use https,
// this function corrects it.
// this needs to check all items to see if they match schema.org, then fix.
func fixContextUrl(jsonld string, ctx string) (string, error) {
	var err error
	contexts := gjson.Get(jsonld, "@context").Map()
	if _, ok := contexts["@vocab"]; !ok {
		jsonld, err = sjson.Set(jsonld, "@context.@vocab", httpsContext)
	}
	// for range
	for ns, c := range contexts {
		var context = c.String()
		if strings.Contains(context, "schema.org") {
			if strings.Contains(context, "www.") { // fix www.schema.org
				var i = strings.Index(context, "schema.org")
				context = context[i:]
				context = ctx + context
			}
			if len(context) < 20 { // https://schema.org/
				context = ctx
			}
		}
		var path = "@context." + ns
		jsonld, err = sjson.Set(jsonld, path, context)
		if err != nil {
			log.Error("Error standardizing schema.org" + err.Error())
		}

	}
	return jsonld, err
}

// Our first json fixup in existence.
// If the top-level JSON-LD context is a string instead of an object,
// this function corrects it.
func fixContextArray(jsonld string, option config.ContextOption) (string, error) {
	var err error
	contexts := gjson.Get(jsonld, "@context")
	switch contexts.Value().(type) {
	case []interface{}: // array
		jsonld, err = standardizeContext(jsonld, config.StandardizedHttps)
	case map[string]interface{}: // array
		// jsonld = jsonld ineffectual assignment commented out
	}
	return jsonld, err
}

// if the top-level JSON-LD @id is not an IRI, and there is no base in the context,
// remove that id
// see https://github.com/piprate/json-gold/discussions/68#discussioncomment-4782788
// for details
func fixId(jsonld string) (string, error) {
	var err error
	originalBase := gjson.Get(jsonld, "@context.@base").String()

	if originalBase != "" { // if we have a context base, there is no need to do any of this
		return jsonld, err
	}
	topLevelType := gjson.Get(jsonld, "@type").String()
	var selector string
	var formatter func(index int) string
	if topLevelType == "Dataset" {
		selector = "@id"
		formatter = func(index int) string { return "@id" }
	} else if topLevelType == "ItemList" {
		selector = "itemListElement.#.item.@id"
		formatter = func(index int) string { return fmt.Sprintf("itemListElement.%v.item.@id", index) }
	} else { // we don't know how to fix any of these other things
		return jsonld, err
	}
	jsonIdentifiers := gjson.Get(jsonld, selector)
	index := 0
	jsonIdentifiers.ForEach(func(key, jsonResult gjson.Result) bool {
		jsonIdentifier := jsonResult.String()
		idUrl, idErr := url.Parse(jsonIdentifier)
		if idUrl.Scheme == "" { // we have a relative url and no base in the context
			jsonld, idErr = sjson.Set(jsonld, formatter(index), "file://"+jsonIdentifier)
		}
		if idErr != nil {
			err = idErr
			return false
		}
		index++
		return true
	})
	return jsonld, err
}

// this just creates a standardized context
// jsonMap := make(map[string]interface{})
var StandardHttpsContext = map[string]interface{}{
	"@vocab": "https://schema.org/",
	"adms":   "https://www.w3.org/ns/adms#",
	"dcat":   "https://www.w3.org/ns/dcat#",
	"dct":    "https://purl.org/dc/terms/",
	"foaf":   "https://xmlns.com/foaf/0.1/",
	"gsp":    "https://www.opengis.net/ont/geosparql#",
	"locn":   "https://www.w3.org/ns/locn#",
	"owl":    "https://www.w3.org/2002/07/owl#",
	"rdf":    "https://www.w3.org/1999/02/22-rdf-syntax-ns#",
	"rdfs":   "https://www.w3.org/2000/01/rdf-schema#",
	"schema": "https://schema.org/",
	"skos":   "https://www.w3.org/2004/02/skos/core#",
	"spdx":   "https://spdx.org/rdf/terms#",
	"time":   "https://www.w3.org/2006/time",
	"vcard":  "https://www.w3.org/2006/vcard/ns#",
	"xsd":    "https://www.w3.org/2001/XMLSchema#",
}

var StandardHttpContext = map[string]interface{}{
	"@vocab": "http://schema.org/",
	"adms":   "http://www.w3.org/ns/adms#",
	"dcat":   "http://www.w3.org/ns/dcat#",
	"dct":    "http://purl.org/dc/terms/",
	"foaf":   "http://xmlns.com/foaf/0.1/",
	"gsp":    "http://www.opengis.net/ont/geosparql#",
	"locn":   "http://www.w3.org/ns/locn#",
	"owl":    "http://www.w3.org/2002/07/owl#",
	"rdf":    "http://www.w3.org/1999/02/22-rdf-syntax-ns#",
	"rdfs":   "http://www.w3.org/2000/01/rdf-schema#",
	"schema": "http://schema.org/",
	"skos":   "http://www.w3.org/2004/02/skos/core#",
	"spdx":   "http://spdx.org/rdf/terms#",
	"time":   "http://www.w3.org/2006/time",
	"vcard":  "http://www.w3.org/2006/vcard/ns#",
	"xsd":    "http://www.w3.org/2001/XMLSchema#",
}

func standardizeContext(jsonld string, option config.ContextOption) (string, error) {

	var err error

	switch option {
	case config.StandardizedHttps:
		jsonld, err = sjson.Set(jsonld, "@context", StandardHttpsContext)
	case config.StandardizedHttp:
		jsonld, err = sjson.Set(jsonld, "@context", StandardHttpContext)
	}
	return jsonld, err
}

// there is a cleaner way to handle this...
func getOptions(ctxOption config.ContextOption) (config.ContextOption, string) {
	ctxString := httpsContext
	if ctxOption != config.Strict {
		if ctxOption == config.Https || ctxOption == config.StandardizedHttps {
			ctxString = httpsContext
		} else {
			ctxString = httpContext
		}
		return config.Https, ctxString
	} else {
		return config.Strict, ctxString
	}

}

// ##### end contxt fixes
func ProcessJson(v1 *viper.Viper,
	source *config.Source, urlloc string, jsonld string) (string, common.Identifier, error) {
	mcfg := v1.GetStringMapString("context")
	var err error
	//sources, err := config.GetSources(v1)
	//source, err := config.GetSourceByName(sources, site)
	srcFixOption, srcHttpOption := getOptions(source.FixContextOption)

	// In the config file, context { strict: true } bypasses these fixups.
	// Strict defaults to false.
	// this is a command level
	if strict, ok := mcfg["strict"]; !(ok && strict == "true") || (srcFixOption != config.Strict) {

		log.Info("context.strict is not set to true; doing json-ld fixups.")
		jsonld, err = fixContextString(jsonld)
		if err != nil {
			log.Error(
				"ERROR: URL: ", urlloc, " Action: Fixing JSON-LD context from string to be an object Error: ", err)
		}
		jsonld, err = fixContextArray(jsonld, srcFixOption)
		if err != nil {
			log.Error("ERROR: URL: ", urlloc, " Action: Fixing JSON-LD context from array to be an object Error: ", err)
		}
		jsonld, err = fixContextUrl(jsonld, srcHttpOption) // CONST for now
		if err != nil {
			log.Error("ERROR: URL: ", urlloc, " Action: Fixing JSON-LD context url scheme and trailing slash Error: ", err)
		}
		jsonld, err = fixId(jsonld)
		if err != nil {
			log.Error("ERROR: URL: ", urlloc, " Action: Removing relative JSON-LD @id Error: ", err)
		}

	}
	identifier, err := common.GenerateFileSha(jsonld)
	if err != nil {
		log.Error("ERROR: URL:", urlloc, "Action: Getting normalized sha  Error:", err)
	}

	return jsonld, identifier, err
}

func Upload(v1 *viper.Viper, mc *minio.Client, bucketName string, site string, urlloc string, jsonld string) (string, error) {

	sources, err := config.GetSources(v1)
	if err != nil {
		return "", err
	}
	source, err := config.GetSourceByName(sources, site)
	if err != nil {
		return "", err
	}

	jsonld, identifier, err := ProcessJson(v1, source, urlloc, jsonld)
	if err != nil {
		return "", err
	}

	sha := identifier.UniqueId
	objectName := fmt.Sprintf("summoned/%s/%s.jsonld", site, sha)
	contentType := JSONContentType
	b := bytes.NewBufferString(jsonld)

	usermeta := make(map[string]string)
	usermeta["url"] = urlloc
	usermeta["sha1"] = sha
	usermeta["uniqueid"] = sha
	usermeta["jsonsha"] = identifier.JsonSha
	usermeta["identifiertype"] = identifier.IdentifierType
	if identifier.MatchedPath != "" {
		usermeta["matchedpath"] = identifier.MatchedPath
		usermeta["matchedstring"] = identifier.MatchedString
	}
	if config.IdentifierString == source.IdentifierType {
		usermeta["sha1"] = identifier.JsonSha
	}
	if source.IdentifierType == config.SourceUrl {
		log.Info("not suppported, yet. needs url sanitizing")
	}
	// write the prov entry for this object
	err = StoreProvNamedGraph(bucketName, mc, site, sha, urlloc, "milled", sources)
	if err != nil {
		return "", err
	}

	// Make sure the object doesn't already exist and we don't accidentally overwrite it
	if _, err := mc.StatObject(context.Background(), bucketName, objectName, minio.StatObjectOptions{}); err == nil {
		return "", err
	}

	_, err = mc.PutObject(context.Background(), bucketName, objectName, b, int64(b.Len()), minio.PutObjectOptions{ContentType: contentType, UserMetadata: usermeta})
	if err != nil {
		return "", err
	}
	log.Debug("Uploaded Bucket:", bucketName, " File:", objectName, "Size", int64(b.Len()))
	return sha, err
}
