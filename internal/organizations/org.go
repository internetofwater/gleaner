package organizations

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"gleaner/internal/config"
	configTypes "gleaner/internal/config"

	log "github.com/sirupsen/logrus"

	"gleaner/internal/common"

	"github.com/minio/minio-go/v7"
	"github.com/spf13/viper"
)

const t = `{
		"@context": {
			"@vocab": "https://schema.org/"
		},
		"@id": "https://gleaner.io/id/org/{{.Name}}",
		"@type": "Organization",
		"url": "{{.URL}}",
		"name": "{{.Name}}",
		 "identifier": {
			"@type": "PropertyValue",
			"@id": "{{.PID}}",
			"propertyID": "https://registry.identifiers.org/registry/doi",
			"url": "{{.PID}}",
			"description": "Persistent identifier for this organization"
		}
	}`

type Qset struct {
	Subject   string `parquet:"name=Subject,  type=BYTE_ARRAY, convertedtype=UTF8, encoding=PLAIN_DICTIONARY"`
	Predicate string `parquet:"name=Predicate,  type=BYTE_ARRAY, convertedtype=UTF8, encoding=PLAIN_DICTIONARY"`
	Object    string `parquet:"name=Object,  type=BYTE_ARRAY, convertedtype=UTF8, encoding=PLAIN_DICTIONARY"`
	Graph     string `parquet:"name=Graph,  type=BYTE_ARRAY, convertedtype=UTF8, encoding=PLAIN_DICTIONARY"`
}

// Make a graph from the Gleaner config file source and
// load this to a /sources bucket
func BuildGraph(mc *minio.Client, v1 *viper.Viper) error {
	// read config file
	bucketName, _ := configTypes.GetBucketName(v1)

	log.Info("Building organization graph.")
	domains, err := config.GetSources(v1)
	if err != nil {
		log.Error(err)
		return err
	}
	jsonldProcessor, options, err := common.GenerateJSONLDProcessor(v1)
	if err != nil {
		return err
	}

	for domainIndex := range domains {

		jsonld, err := buildOrgJSONLD(domains[domainIndex])
		if err != nil {
			return err
		}

		rdfRepresentation, err := common.JLD2nq(jsonld, jsonldProcessor, options)
		if err != nil {
			return err
		}

		rdfBuffer := bytes.NewBufferString(rdfRepresentation)

		objectName := fmt.Sprintf("orgs/%s.nq", domains[domainIndex].Name)

		if _, err := mc.PutObject(context.Background(), bucketName, objectName, rdfBuffer, int64(rdfBuffer.Len()), minio.PutObjectOptions{ContentType: "application/ld+json"}); err != nil {
			return err
		}
	}

	return nil
}

// build the JSONLD document that associated the crawl with its organizational data
func buildOrgJSONLD(src config.Sources) (string, error) {
	var doc bytes.Buffer

	template, err := template.New("prov").Parse(t)
	if err != nil {
		return "", err
	}

	if err := template.Execute(&doc, src); err != nil {
		return "", err
	}

	return doc.String(), err
}
