package config

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// auth fails if a region is set in minioclient...
// frig frig... do not use lowercase... those are private variables
type Minio struct {
	Address   string // `mapstructure:"MINIO_ADDRESS"`
	Port      int    //`mapstructure:"MINIO_PORT"`
	Ssl       bool   //`mapstructure:"MINIO_USE_SSL"`
	Bucket    string
	Region    string
	Accesskey string //`mapstructure:"MINIO_ACCESS_KEY"`
	Secretkey string // `mapstructure:"MINIO_SECRET_KEY"`

}

// auth fails if a region is set in minioclient...
var MinioTemplate = map[string]interface{}{
	"minio": map[string]string{
		"address":   "localhost",
		"port":      "9000",
		"bucket":    "",
		"ssl":       "false",
		"region":    "",
		"accesskey": "",
		"secretkey": "",
	},
}

// use config.Sub("minio)
func ReadMinioConfig(minioSubtree *viper.Viper) (Minio, error) {
	var minioCfg Minio
	for key, value := range MinioTemplate {
		minioSubtree.SetDefault(key, value)
	}

	err := minioSubtree.Unmarshal(&minioCfg)
	if err != nil {
		log.Fatal("error when parsing minio config: ", err)
	}
	return minioCfg, err
}

func GetBucketName(v1 *viper.Viper) (string, error) {
	minSubtree := v1.Sub("minio")
	miniocfg, err := ReadMinioConfig(minSubtree)
	if err != nil {
		log.Fatal("Cannot read bucket name from configuration/minio")

	}
	bucketName := miniocfg.Bucket //miniocfg["bucket"] //   get the top level bucket for all of gleaner operations from config file
	return bucketName, err
}
