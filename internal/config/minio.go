package config

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// auth fails if a region is set in minioclient...
type Minio struct {
	Address   string // `mapstructure:"MINIO_ADDRESS"`
	Port      int    //`mapstructure:"MINIO_PORT"`
	Ssl       bool   //`mapstructure:"MINIO_USE_SSL"`
	Bucket    string
	Region    string
	Accesskey string //`mapstructure:"MINIO_ACCESS_KEY"`
	Secretkey string // `mapstructure:"MINIO_SECRET_KEY"`

}

var MinioTemplate = map[string]interface{}{
	"minio": map[string]string{
		"address":   "localhost",
		"port":      "9000",
		"bucket":    "",
		"ssl":       "false",
		"region":    "", // auth fails if a region is set in minioclient
		"accesskey": "",
		"secretkey": "",
	},
}

// Read the minio section from the viper config
func ReadMinioConfig(minioSubtree *viper.Viper) (Minio, error) {
	var minioCfg Minio
	for key, value := range MinioTemplate {
		minioSubtree.SetDefault(key, value)
	}

	err := minioSubtree.Unmarshal(&minioCfg)
	if err != nil {
		return minioCfg, fmt.Errorf("error when parsing minio endpoint config: %v", err)
	}
	return minioCfg, err
}

// Gets the name of the minio bucket specified in the gleaner config
func GetBucketName(v1 *viper.Viper) (string, error) {
	minSubtree := v1.Sub("minio")
	miniocfg, err := ReadMinioConfig(minSubtree)
	if err != nil {
		log.Fatal("Cannot read bucket name from configuration/minio")

	}
	return miniocfg.Bucket, err
}
