package organizations

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	configTypes "github.com/gleanerio/gleaner/internal/config"
	log "github.com/sirupsen/logrus"

	"github.com/gleanerio/gleaner/internal/common"
	"github.com/gleanerio/gleaner/internal/objects"

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

// Mmake a graph from the Gleaner config file source and
// load this to a /sources bucket
func BuildGraph(mc *minio.Client, v1 *viper.Viper) error {
	//var (
	//	buf    bytes.Buffer
	//	logger = log.New(&buf, "logger: ", log.Lshortfile)
	//)

	// read config file
	//miniocfg := v1.GetStringMapString("minio")
	//bucketName := miniocfg["bucket"] //   get the top level bucket for all of gleaner operations from config file
	bucketName, _ := configTypes.GetBucketName(v1)

	log.Info("Building organization graph.")
	domains := objects.SourcesAndGraphs(v1)
	proc, options := common.JLDProc(v1)

	// Sources: Name, Logo, URL, Headless, Pid
	for domainIndex := range domains {

		// log.Println(domains[k])

		jld, err := orggraph(domains[domainIndex])
		if err != nil {
			log.Error(err)
			return err
		}

		rdfRepresentation, err := common.JLD2nq(jld, proc, options)
		if err != nil {
			log.Error(err)
			return err
		}

		rdfBuffer := bytes.NewBufferString(rdfRepresentation)

		// load to minio
		// orgsha := common.GetSHA(jld)
		// objectName := fmt.Sprintf("orgs/%s/%s.nq", domains[k].Name, orgsha) // k is the name of the provider from config
		objectName := fmt.Sprintf("orgs/%s.nq", domains[domainIndex].Name) // k is the name of the provider from config
		contentType := "application/ld+json"

		// Upload the file with FPutObject
		_, err = mc.PutObject(context.Background(), bucketName, objectName, rdfBuffer, int64(rdfBuffer.Len()), minio.PutObjectOptions{ContentType: contentType})
		if err != nil {
			log.Error(objectName, err)
			return err
		}

	}

	return nil
}

func orggraph(src objects.Sources) (string, error) {
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
