package check

import (
	"context"
	"fmt"

	"gleaner/internal/config"

	"github.com/minio/minio-go/v7"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func isExists(bucketName string, buckets []minio.BucketInfo) (exists bool) {

	for _, search := range buckets {
		if search.Name == bucketName {
			return true
		}
	}
	return false
}

// Buckets checks the setup
func validateBuckets(mc *minio.Client, bucket string) error {
	var err error

	buckets, err := mc.ListBuckets(context.Background())
	found := isExists(bucket, buckets)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("unable to locate required bucket:  %s, did you run gleaner with -setup the first to set up buckets?", bucket)
	}
	if found {
		log.Debug("Validated access to object store:", bucket)
	}

	return err
}

// MakeBuckets checks the setup
func MakeBuckets(mc *minio.Client, bucket string) error {

	found, err := mc.BucketExists(context.Background(), bucket)
	if err != nil {
		return err
	}
	if found {
		log.Debug("Gleaner Bucket", bucket, "found.")
	} else {
		log.Debug("Gleaner Bucket", bucket, "not found, generating")
		err = mc.MakeBucket(context.Background(), bucket, minio.MakeBucketOptions{Region: "us-east-1"}) // location is kinda meaningless here
		if err != nil {
			log.Debug("Make bucket:", err)
			return err
		}
	}

	return err
}

// Setup Gleaner buckets
func Setup(mc *minio.Client, v1 *viper.Viper) error {
	ms := v1.Sub("minio")
	m1, err := config.ReadMinioConfig(ms)
	if err != nil {
		log.Error("Error reading gleaner config", err)
		return err
	}
	log.Info("Validating access to object store")

	// Check if we can connect, don't care about buckets at this point
	_, err = mc.ListBuckets(context.Background())

	if err != nil {
		log.Error("Connection issue, make sure the minio server is running and accessible.", err)
		return err
	}
	// If requested, set up the buckets
	log.Info("Setting up buckets")
	err = MakeBuckets(mc, m1.Bucket)
	if err != nil {
		log.Error("Error making buckets for Setup call", err)
		return err
	}

	err = PreflightChecks(mc, v1) // postsetup test ;)
	if err != nil {
		return err
	}
	log.Info("Buckets generated.  Object store should be ready for runs")
	return nil

}

// Check if we can connect and that the proper bucket exists
func PreflightChecks(mc *minio.Client, v1 *viper.Viper) error {
	// Validate Minio access
	bucketName, err := config.GetBucketName(v1)

	if err != nil {
		log.Error("missing bucket name.", err)
		return err
	}

	//Check our bucket is ready
	err = validateBuckets(mc, bucketName)
	if err != nil {
		log.Error("Can not find bucket.", err)
		return err
	}
	return nil
}
