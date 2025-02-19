package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"

	log "github.com/sirupsen/logrus"
)

type GleanerConfig struct {
	Minio       MinioConfig
	Context     ContextConfig
	ContextMaps []ContextMap
	Sources     []SourceConfig
	Rude        bool
	Mode        string // "full" or "diff"
}

func (gc GleanerConfig) WriteConfigAs(filePath string) error {

	data, err := yaml.Marshal(gc)
	if err != nil {
		return err
	}

	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return err
	}

	return nil
}

func (gc GleanerConfig) GetHeadlessSources() []SourceConfig {
	var headlessSources []SourceConfig
	for _, source := range gc.Sources {
		if source.Headless {
			headlessSources = append(headlessSources, source)
		}
	}
	return headlessSources
}

type MinioConfig struct {
	Address   string
	Port      int
	Ssl       bool
	Accesskey string
	Secretkey string
	Bucket    string
	Region    string
}

func (mcfg MinioConfig) NewClient() (*minio.Client, error) {
	var endpoint, accessKeyID, secretAccessKey string
	var useSSL bool

	if mcfg.Port == 0 {
		endpoint = mcfg.Address
		accessKeyID = mcfg.Accesskey
		secretAccessKey = mcfg.Secretkey
		useSSL = mcfg.Ssl
	} else {
		endpoint = fmt.Sprintf("%s:%d", mcfg.Address, mcfg.Port)
		accessKeyID = mcfg.Accesskey
		secretAccessKey = mcfg.Secretkey
		useSSL = mcfg.Ssl
	}

	var minioClient *minio.Client
	var err error

	if mcfg.Region == "" {
		log.Warn("no region set")
		opts := &minio.Options{
			Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
			Secure: useSSL,
		}
		minioClient, err = minio.New(endpoint, opts)
		if err != nil {
			return nil, err
		}
	} else {
		log.Warn("region set; for GCS or AWS, may cause issues with minio")
		region := mcfg.Region
		opts := &minio.Options{
			Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
			Secure: useSSL,
			Region: region,
		}
		minioClient, err = minio.New(endpoint, opts)
		if err != nil {
			return nil, err
		}
	}

	return minioClient, nil

}

type ContextConfig struct {
	Cache  bool
	Strict bool
}

type ContextMap struct {
	Prefix string
	File   string
}

type SourceConfig struct {
	Domain     string
	Name       string
	Sourcetype string
	Url        string
	Headless   bool
}

// ensures all struct fields are present in the YAML config and errors if any are missing
func checkMissingFields(v *viper.Viper, structType reflect.Type, parentKey string) error {
	var missingFields []string

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		fieldName := field.Tag.Get("mapstructure")
		if fieldName == "" {
			fieldName = strings.ToLower(field.Name) // Default to lowercase field name
		}

		fullKey := fieldName
		if parentKey != "" {
			fullKey = parentKey + "." + fieldName
		}

		if field.Type.Kind() == reflect.Struct {
			// Recursively check nested structs
			if err := checkMissingFields(v, field.Type, fullKey); err != nil {
				missingFields = append(missingFields, err.Error())
			}
		} else if !v.IsSet(fullKey) {
			missingFields = append(missingFields, fullKey)
		}
	}

	if len(missingFields) > 0 {
		return fmt.Errorf("missing required fields: %v", strings.Join(missingFields, ", "))
	}

	return nil
}

func ReadGleanerConfig(cfgPath, filename string) (GleanerConfig, error) {
	v := viper.New()
	nameWithoutExt := strings.TrimSuffix(filename, filepath.Ext(filename))
	v.SetConfigName(nameWithoutExt)
	v.AddConfigPath(cfgPath)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return GleanerConfig{}, err
	}

	// Check for missing required fields before unmarshaling
	if err := checkMissingFields(v, reflect.TypeOf(GleanerConfig{}), ""); err != nil {
		return GleanerConfig{}, err
	}

	var config GleanerConfig
	if err := v.UnmarshalExact(&config); err != nil {
		return GleanerConfig{}, err
	}

	return config, nil
}
