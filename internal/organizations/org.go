package organizations

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"gleaner/cmd/config"

	log "github.com/sirupsen/logrus"

	"gleaner/internal/common"

	"github.com/minio/minio-go/v7"
)

// Represents the JSONLD file that will be converted into an nq for each org in the org/ bucket
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
func SummonOrgs(mc *minio.Client, conf config.GleanerConfig) error {

	log.Info("Building organization graph.")
	jsonldProcessor, options, err := common.GenerateJSONLDProcessor(conf)
	if err != nil {
		return err
	}

	for _, domain := range conf.Sources {

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

		if _, err := mc.PutObject(context.Background(), conf.Minio.Bucket, objectName, rdfBuffer, int64(rdfBuffer.Len()), minio.PutObjectOptions{ContentType: "application/n-quads"}); err != nil {
			return err
		}
	}

	return nil
}

// Create a provenance ontology JSONLD string that associates the crawl with its organizational data
func BuildOrgJSONLD(src config.SourceConfig) (string, error) {

	// Make sure there are no empty string values for fields that would be
	// inserted into the jsonld template
	for _, field := range []struct {
		name string
		val  string
	}{
		{"Name", src.Name},
		{"URL", src.Url},
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
