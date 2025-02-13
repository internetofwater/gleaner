package internal

import (
	"bytes"
	"context"
	"io"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/minio/minio-go/v7"
)

// ProvURL returns the URLs we have currently indexed as recorded in the prov records
func GetProvURLs(minioClient *minio.Client, bucket, prefix string) []string {

	opts := minio.SelectObjectOptions{
		Expression:     "select s.\"@graph\"[1]['@id'] from s3object s",
		ExpressionType: minio.QueryExpressionTypeSQL,
		InputSerialization: minio.SelectObjectInputSerialization{
			CompressionType: minio.SelectCompressionNONE,
			JSON: &minio.JSONInputOptions{
				Type: minio.JSONDocumentType,
			},
		},
		OutputSerialization: minio.SelectObjectOutputSerialization{
			CSV: &minio.CSVOutputOptions{
				RecordDelimiter: "\n",
				FieldDelimiter:  ",",
			},
		},
	}

	semaphoreChan := make(chan struct{}, 20) // a blocking channel to keep concurrency under control (1 == single thread)
	defer close(semaphoreChan)
	wg := sync.WaitGroup{} // a wait group enables the main process a wait for goroutines to finish

	// params for list objects calls
	doneCh := make(chan struct{}) // , N) Create a done channel to control 'ListObjectsV2' go routine.
	defer close(doneCh)           // Indicate to our routine to exit cleanly upon return.
	// isRecursive := true

	oa := []string{}

	for object := range minioClient.ListObjects(context.Background(), bucket,
		minio.ListObjectsOptions{Prefix: prefix, Recursive: true}) {

		wg.Add(1)
		go func(object minio.ObjectInfo) {

			log.Trace("Bucket", bucket, "object:", object.Key)

			reader, err := minioClient.SelectObjectContent(context.Background(), bucket, object.Key, opts)
			if err != nil {
				log.Fatal(err)
			}
			defer reader.Close()

			buf := bytes.NewBufferString("")
			if _, err := io.Copy(buf, reader); err != nil {
				log.Fatal(err)
			}

			oa = append(oa, strings.TrimSpace(buf.String()))

			wg.Done() // tell the wait group that we be done
			log.Trace("Doc:", object.Key, "error", err)
			<-semaphoreChan

		}(object)
		wg.Wait()

	}

	log.Info("bucket", bucket, ":", prefix, "object count:", len(oa))

	return oa
}
