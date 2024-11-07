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

const orgJSONLDTemplate = `{
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

// For each source in the gleaner config, build the JSONLD for the org,
// convert that to nq, and upload to minio
func BuildGraph(mc *minio.Client, v1 *viper.Viper) error {

	bucketName, err := configTypes.GetBucketName(v1)
	if err != nil {
		return err
	}

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

	for _, domain := range domains {

		jsonld, err := BuildOrgJSONLD(domain)
		if err != nil {
			return err
		}

		rdfRepresentation, err := common.JLD2nq(jsonld, jsonldProcessor, options)
		if err != nil {
			return err
		}

		rdfBuffer := bytes.NewBufferString(rdfRepresentation)

		objectName := fmt.Sprintf("orgs/%s.nq", domain.Name)

		if _, err := mc.PutObject(context.Background(), bucketName, objectName, rdfBuffer, int64(rdfBuffer.Len()), minio.PutObjectOptions{ContentType: "application/ld+json"}); err != nil {
			return err
		}
	}

	return nil
}

// build the provenance ontology JSONLD document that associates the crawl with its organizational data
func BuildOrgJSONLD(src config.Source) (string, error) {

	// Make sure there are no empty string values for fields that would be
	// inserted into the jsonld template
	for _, field := range []struct {
		name string
		val  string
	}{
		{"PID", src.PID},
		{"Name", src.Name},
		{"URL", src.URL},
	} {
		if field.val == "" {
			return "", fmt.Errorf("source %s is missing required field %s", src.Name, field.name)
		}
	}

	template, err := template.New("prov").Option("missingkey=error").Parse(orgJSONLDTemplate)
	if err != nil {
		return "", err
	}
	var jsonldBuffer bytes.Buffer

	if err := template.Execute(&jsonldBuffer, src); err != nil {
		return "", err
	}

	return jsonldBuffer.String(), err
}
