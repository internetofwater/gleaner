package common

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinioConnection Set up minio and initialize client
func MinioConnection(port int, address, secretKey, accessKey string, region string, ssl bool) (*minio.Client, error) {

	var endpoint string
	if port == 0 {
		endpoint = address
	} else {
		endpoint = fmt.Sprintf("%s:%d", address, port)
	}

	var minioClient *minio.Client
	var err error

	if region == "" {
		log.Debug("no region set")
		minioClient, err = minio.New(endpoint,
			&minio.Options{Creds: credentials.NewStaticV4(accessKey, secretKey, ""),
				Secure: ssl,
			})
	} else {
		minioClient, err = minio.New(endpoint,
			&minio.Options{Creds: credentials.NewStaticV4(accessKey, secretKey, ""),
				Secure: ssl,
				Region: region,
			})
	}
	return minioClient, err

}
