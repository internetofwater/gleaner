package internal

import (
	"context"
	"fmt"
	"gleaner/cmd/config"

	"github.com/minio/minio-go/v7"
	log "github.com/sirupsen/logrus"
)

// Buckets checks the setup
func validateBuckets(mc *minio.Client, bucket string) error {

	buckets, err := mc.ListBuckets(context.Background())
	if err != nil {
		return err
	}
	var found = false
	for _, b := range buckets {
		if b.Name == bucket {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("unable to locate required bucket:  %s, did you run gleaner with --setup the first to set up buckets?", bucket)
	} else {
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
func Setup(mc *minio.Client, conf config.MinioConfig) error {
	log.Info("Validating access to object store")

	// Check if we can connect, don't care about buckets at this point
	_, err := mc.ListBuckets(context.Background())

	if err != nil {
		log.Error("Connection issue, make sure the minio server is running and accessible.", err)
		return err
	}
	// If requested, set up the buckets
	log.Info("Setting up buckets")
	err = MakeBuckets(mc, conf.Bucket)
	if err != nil {
		log.Error("Error making buckets for Setup call", err)
		return err
	}
	//Check our bucket is ready
	err = validateBuckets(mc, conf.Bucket)
	if err != nil {
		log.Error("Can not find bucket.", err)
		return err
	}

	log.Info("Buckets generated.  Object store should be ready for runs")
	return nil

}
