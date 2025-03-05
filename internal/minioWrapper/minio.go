package minioWrapper

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioClientWrapper struct {
	DefaultBucket string
	Client        *minio.Client
}

// NewMinioConnection Set up minio and initialize client
func NewMinioConnection(port int, address, secretKey, accessKey string, region string, ssl bool, defaultBucket string) (MinioClientWrapper, error) {

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
	return MinioClientWrapper{Client: minioClient, DefaultBucket: defaultBucket}, err

}

func (m *MinioClientWrapper) SetupBucket() error {
	if exists, err := m.Client.BucketExists(context.Background(), m.DefaultBucket); err != nil {
		return err
	} else if !exists {
		if err := m.Client.MakeBucket(context.Background(), m.DefaultBucket, minio.MakeBucketOptions{}); err != nil {
			return err
		}
	}

	return m.validateBucket()
}

func (m *MinioClientWrapper) validateBucket() error {
	if exists, err := m.Client.BucketExists(context.Background(), m.DefaultBucket); err != nil {
		return err
	} else if !exists {
		return fmt.Errorf("bucket %s does not exist", m.DefaultBucket)
	}
	return nil
}
