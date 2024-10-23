package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gleanerio/gleaner/internal/check"
	"github.com/gleanerio/gleaner/internal/common"
	"github.com/gleanerio/gleaner/internal/config"
	"github.com/gleanerio/gleaner/internal/millers"
	"github.com/gleanerio/gleaner/internal/objects"
	"github.com/gleanerio/gleaner/internal/organizations"
	"github.com/gleanerio/gleaner/internal/summoner"
	"github.com/gleanerio/gleaner/internal/summoner/acquire"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var addressVal, portVal, bucketVal, sourceVal, configVal, modeVal, secretKeyVal, accessKeyVal string
var sslVal, setupBucketsVal, rudeVal bool

// Entrypoint for the gleaner command
func Gleaner() error {

	v1, err := config.ReadGleanerConfig(filepath.Base(configVal), filepath.Dir(configVal))
	if err != nil {
		return fmt.Errorf("error when reading config: %v", err)
	}
	if v1.Sub("minio") == nil {
		return errors.New("no minio config after reading config")
	}

	if sourceVal != "" {
		tmp := []objects.Sources{} // tmp slice to hold our desired source

		var domains []objects.Sources
		err := v1.UnmarshalKey("sources", &domains)
		if err != nil {
			log.Warn(err)
		}

		for _, k := range domains {
			if sourceVal == k.Name {
				k.Active = true
				tmp = append(tmp, k)
			}
		}

		if len(tmp) == 0 {
			return fmt.Errorf("no matching source, did your --source VALUE match a sources.name value in %s", configVal)
		}

		// Replace the soures in the config with the one we specified
		configMap := v1.AllSettings()
		delete(configMap, "sources")
		v1.Set("sources", tmp)

		if rudeVal {
			v1.Set("rude", true)
		}
	} else if rudeVal && sourceVal == "" {
		return errors.New("rude is only valid when --source is also specified")
	}

	// Parse a new mode entry from command line if present
	if modeVal != "" {
		m := v1.GetStringMap("summoner")
		m["mode"] = modeVal
		v1.Set("summoner", m)
	}
	if addressVal != "" {
		minio_config := v1.GetStringMap("minio")
		minio_config["address"] = addressVal
		v1.Set("minio", minio_config)
	}
	if secretKeyVal != "" {
		minio_config := v1.GetStringMap("minio")
		minio_config["secretkey"] = secretKeyVal
		v1.Set("minio", minio_config)
	}
	if portVal != "" {
		minio_config := v1.GetStringMap("minio")
		minio_config["port"] = portVal
		v1.Set("minio", minio_config)
	}

	if v1.Sub("minio") == nil {
		return errors.New("no minio config after applying args")
	}
	// Set up the minio connector
	mc := common.MinioConnection(v1)

	// If requested, set up the buckets
	if setupBucketsVal {
		log.Info("Setting up buckets inside minio")
		err = check.Setup(mc, v1)
		if err != nil {
			return errors.New("error making buckets for setup call")
		}
		log.Info("Buckets generated. Object store should be ready for runs")
	}

	// Validate Minio access
	err = check.PreflightChecks(mc, v1)
	if err != nil {
		return fmt.Errorf("minio access check failed. Make sure the server is running. Full error was: '%v'", err)
	}

	mcfg := v1.GetStringMapString("gleaner")

	// err := organizations.BuildGraphMem(mc, v1) // parfquet testing
	if err := organizations.BuildGraph(mc, v1); err != nil {
		return err
	}

	// If configured, summon sources
	if mcfg["summon"] == "true" {
		// Index the sitegraphs first, if any
		fn, err := acquire.GetGraph(mc, v1)
		if err != nil {
			log.Error(err)
		}
		log.Info(fn)
		// summon sitemaps
		summoner.Summoner(mc, v1)
	}

	// if configured, process summoned sources fronm JSON-LD to RDF (nq)
	if mcfg["mill"] == "true" {
		millers.Millers(mc, v1) // need to remove rundir and then fix the compile
	}
	return err
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:              "gleaner",
	TraverseChildren: true,
	Short:            "Extract JSON-LD from web pages exposed in a domains sitemap file.",
	Run: func(cmd *cobra.Command, args []string) {
		err := Gleaner()
		if err != nil {
			log.Fatal(err)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	akey := os.Getenv("MINIO_ACCESS_KEY")
	skey := os.Getenv("MINIO_SECRET_KEY")
	if skey != "" || akey != "" {
		fmt.Println(" MINIO_ACCESS_KEY or  MINIO_SECRET_KEY are set")
		fmt.Println("if this is not intentional, please unset")
	}

	// Persistent flags defined here will be global for the entire application.
	rootCmd.PersistentFlags().StringVar(&configVal, "cfg", "", "compatibility/overload: full path to config file (default location gleaner in configs/local)")
	rootCmd.PersistentFlags().StringVar(&sourceVal, "source", "", "source name")
	rootCmd.PersistentFlags().StringVar(&modeVal, "mode", "local", "Set the mode (full | diff) to index all or just diffs")
	rootCmd.PersistentFlags().StringVar(&addressVal, "address", "localhost", "FQDN for server")
	rootCmd.PersistentFlags().StringVar(&portVal, "port", "9000", "Port for minio server")
	rootCmd.PersistentFlags().StringVar(&bucketVal, "bucket", "gleaner", "The configuration bucket")
	rootCmd.PersistentFlags().StringVar(&accessKeyVal, "accesskey", "", "Minio access key")
	rootCmd.PersistentFlags().StringVar(&secretKeyVal, "secretkey", "", "Minio secret key")

	rootCmd.PersistentFlags().BoolVar(&sslVal, "ssl", false, "Use SSL when connecting to minio")
	rootCmd.PersistentFlags().BoolVar(&rudeVal, "rude", false, "Ignore robots.txt when connecting to source")
	rootCmd.PersistentFlags().BoolVar(&setupBucketsVal, "setup", false, "Setup buckets in minio")

	cobra.OnInitialize(common.InitLogging)
}
