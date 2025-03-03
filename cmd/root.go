package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gleaner/internal/check"
	"gleaner/internal/common"
	"gleaner/internal/config"
	"gleaner/internal/organizations"
	"gleaner/internal/summoner/acquire"

	"github.com/minio/minio-go/v7"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type GleanerClient struct {
	Address      string // address for minio
	Port         string // port for minio
	Bucket       string // minio bucket to put data
	Source       string // source to crawl from the config
	Config       string // full path to config
	Mode         string
	SecretKey    string // secret key for minio
	AccessKey    string // access key for minio
	SSL          bool   // use SSL for HTTP requests
	SetupBuckets bool   // setup buckets before crawling
	Rude         bool   // ignore robots.txt
}

// Summoner pulls the resources from the data facilities
func (g *GleanerClient) summon(mc *minio.Client, v1 *viper.Viper) error {

	start := time.Now()

	// Get a list of resource URLs that do and don't require headless processing
	domainToUrls, err := acquire.ResourceURLs(v1, mc, false)
	if err != nil {
		log.Error("Error getting urls that do not require headless processing:", err)
		return err
	}
	// just report the error, and then run gathered urls
	if len(domainToUrls) > 0 {
		acquire.ResRetrieve(v1, mc, domainToUrls) // TODO  These can be go funcs that run all at the same time..
	}

	hru, err := acquire.ResourceURLs(v1, mc, true)
	if err != nil {
		log.Error("Error getting urls that require headless processing:", err)
		return err
	}
	// just report the error, and then run gathered urls
	if len(hru) > 0 {
		log.Info("running headless:")
		acquire.HeadlessNG(v1, mc, hru)
	}

	log.Infof("Summoner took %f minutes to run", time.Since(start).Minutes())

	return err
}

// Entrypoint for the gleaner command
func (cli *GleanerClient) Run() error {
	v1, err := config.ReadGleanerConfig(filepath.Base(cli.Config), filepath.Dir(cli.Config))
	if err != nil {
		return fmt.Errorf("error when reading config: %v", err)
	}
	if v1.Sub("minio") == nil {
		return errors.New("no minio config after reading config")
	}

	if cli.Source != "" {
		requestedSources := []config.Source{} // tmp slice to hold our desired source

		var domains []config.Source
		err := v1.UnmarshalKey("sources", &domains)
		if err != nil {
			log.Warn(err)
		}

		for _, k := range domains {
			if cli.Source == k.Name {
				k.Active = true
				requestedSources = append(requestedSources, k)
			}
		}

		if len(requestedSources) == 0 {
			return fmt.Errorf("no matching source, did your --source VALUE match a sources.name value in %s", cli.Config)
		}

		// Replace the soures in the config with the subset we specified
		configMap := v1.AllSettings()
		delete(configMap, "sources")
		v1.Set("sources", requestedSources)

		if cli.Rude {
			v1.Set("rude", true)
		}
	} else if cli.Rude && cli.Source == "" {
		return errors.New("rude is only valid when --source is also specified")
	}

	// Parse a new mode entry from command line if present
	if cli.Mode != "" {
		m := v1.GetStringMap("summoner")
		m["mode"] = cli.Mode
		v1.Set("summoner", m)
	}
	if cli.Address != "" {
		minio_config := v1.GetStringMap("minio")
		minio_config["address"] = cli.Address
		v1.Set("minio", minio_config)
	}
	if cli.SecretKey != "" {
		minio_config := v1.GetStringMap("minio")
		minio_config["secretkey"] = cli.SecretKey
		v1.Set("minio", minio_config)
	}
	if cli.Port != "" {
		minio_config := v1.GetStringMap("minio")
		minio_config["port"] = cli.Port
		v1.Set("minio", minio_config)
	}

	if v1.Sub("minio") == nil {
		return errors.New("no minio config after applying args")
	}
	// Set up the minio connector
	mc := common.MinioConnection(v1)

	// If requested, set up the buckets
	if cli.SetupBuckets {
		log.Info("Setting up buckets inside minio")
		err = check.Setup(mc, v1)
		if err != nil {
			return errors.New("error making buckets for setup call")
		}
	}

	// idate Minio access
	err = check.PreflightChecks(mc, v1)
	if err != nil {
		return fmt.Errorf("minio access check failed. Make sure the server is running. Full error was: '%v'", err)
	}

	gleanerCfgSection := v1.GetStringMapString("gleaner")
	if gleanerCfgSection == nil {
		return errors.New("the 'gleaner' section in " + cli.Config + " is missing")
	}

	if err := organizations.BuildOrgNqsAndUpload(mc, v1); err != nil {
		return err
	}

	if gleanerCfgSection["summon"] == "true" {

		if err := cli.summon(mc, v1); err != nil {
			return err
		}
	}

	return err
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:              "gleaner",
	TraverseChildren: true,
	Short:            "Extract JSON-LD from web pages exposed in a domains sitemap file.",
	Run: func(cmd *cobra.Command, args []string) {

		gleanerCliArgs := &GleanerClient{}
		gleanerCliArgs.Address, _ = cmd.Flags().GetString("address")
		gleanerCliArgs.Port, _ = cmd.Flags().GetString("port")
		gleanerCliArgs.Bucket, _ = cmd.Flags().GetString("bucket")
		gleanerCliArgs.Source, _ = cmd.Flags().GetString("source")
		gleanerCliArgs.Config, _ = cmd.Flags().GetString("cfg")
		gleanerCliArgs.Mode, _ = cmd.Flags().GetString("mode")
		gleanerCliArgs.SecretKey, _ = cmd.Flags().GetString("secretkey")
		gleanerCliArgs.AccessKey, _ = cmd.Flags().GetString("accesskey")
		gleanerCliArgs.SSL, _ = cmd.Flags().GetBool("ssl")
		gleanerCliArgs.SetupBuckets, _ = cmd.Flags().GetBool("setup")
		gleanerCliArgs.Rude, _ = cmd.Flags().GetBool("rude")

		logLevel, _ := cmd.Flags().GetString("log-level")

		switch logLevel {
		case "DEBUG":
			log.SetLevel(log.DebugLevel)
		case "INFO":
			log.SetLevel(log.InfoLevel)
		case "WARN":
			log.SetLevel(log.WarnLevel)
		case "ERROR":
			log.SetLevel(log.ErrorLevel)
		case "FATAL":
			log.SetLevel(log.FatalLevel)
		default:
			log.Fatalf("Invalid log level: %s", logLevel)
		}
		log.SetFormatter(&log.JSONFormatter{})

		if err := gleanerCliArgs.Run(); err != nil {
			log.Fatal(err)
		}
	},
}

// Adds all child commands to the root command and sets flags appropriately.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	akey := os.Getenv("MINIO_ACCESS_KEY")
	skey := os.Getenv("MINIO_SECRET_KEY")
	if skey != "" || akey != "" {
		fmt.Println(" MINIO_ACCESS_KEY or MINIO_SECRET_KEY are set.")
		fmt.Println("if this is not intentional, please unset")
	}
	// Persistent flags defined here will be global for the entire application.
	rootCmd.PersistentFlags().String("cfg", "", "compatibility/overload: full path to config file (default location gleaner in configs/local)")
	rootCmd.PersistentFlags().String("source", "", "source name")
	rootCmd.PersistentFlags().String("mode", "local", "Set the mode (full | diff) to index all or just diffs")
	rootCmd.PersistentFlags().String("address", "", "FQDN for server")
	rootCmd.PersistentFlags().String("port", "", "Port for minio server")
	rootCmd.PersistentFlags().String("bucket", "", "The bucket in which to place gleaner objects")
	rootCmd.PersistentFlags().String("accesskey", "", "Minio access key")
	rootCmd.PersistentFlags().String("secretkey", "", "Minio secret key")
	rootCmd.PersistentFlags().Bool("ssl", false, "Use SSL when connecting to minio")
	rootCmd.PersistentFlags().Bool("rude", false, "Ignore robots.txt when connecting to source")
	rootCmd.PersistentFlags().Bool("setup", false, "Setup buckets in minio")
	rootCmd.PersistentFlags().String("log-level", "INFO", "the log level to use for the nabu logger")
}
