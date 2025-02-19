package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"gleaner/cmd/config"
	"gleaner/internal"
	"gleaner/internal/common"
	"gleaner/internal/organizations"
	"gleaner/internal/summoner"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type GleanerCliArgs struct {
	Address      string // address for minio
	Port         string // port for minio
	Bucket       string // minio bucket to put data
	Source       string // source to crawl from the config
	Config       string // full path to config
	Mode         string // download all or just new data (full | diff)
	SecretKey    string // secret key for minio
	AccessKey    string // access key for minio
	SSL          bool   // use SSL for HTTP requests
	SetupBuckets bool   // setup buckets before crawling
	Rude         bool   // ignore robots.txt
}

// Entrypoint for the gleaner command. Cli args take priority over config
func Gleaner(cli *GleanerCliArgs, conf config.GleanerConfig) error {

	if cli.Source != "" {
		var found bool = false
		for _, s := range conf.Sources {
			if s.Name == cli.Source {
				found = true
				conf.Sources = []config.SourceConfig{s}
				break
			}
		}
		if !found {
			return fmt.Errorf("source %s not found in config", cli.Source)
		}
	}

	if cli.Rude && cli.Source == "" {
		return errors.New("rude is only valid when --source is also specified")
	}

	if cli.Address != "" {
		conf.Minio.Address = cli.Address
	}
	if cli.SecretKey != "" {
		conf.Minio.Secretkey = cli.SecretKey
	}
	if cli.Port != "" {
		portAsInt, err := strconv.Atoi(cli.Port)
		if err != nil {
			return err
		}
		conf.Minio.Port = portAsInt
	}

	mc, err := conf.Minio.NewClient()
	if err != nil {
		return fmt.Errorf("error creating minio client: %v", err)
	}

	// If requested, set up the buckets
	if cli.SetupBuckets {
		log.Info("Setting up buckets inside minio")
		if err := internal.Setup(mc, conf.Minio); err != nil {
			log.Error("error making buckets for Setup call", err)
			return err
		}
	}

	if err := organizations.SummonOrgs(mc, conf); err != nil {
		return err
	}

	if err := summoner.SummonSitemaps(mc, conf); err != nil {
		return err
	}

	return err
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:              "gleaner",
	TraverseChildren: true,
	Short:            "Extract JSON-LD from web pages exposed in a domains sitemap file.",
	Run: func(cmd *cobra.Command, args []string) {

		gleanerCliArgs := &GleanerCliArgs{}
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

		conf, err := config.ReadGleanerConfig(filepath.Dir(gleanerCliArgs.Config), filepath.Base(gleanerCliArgs.Config))
		if err != nil {
			log.Fatal(err)
		}

		if err := Gleaner(gleanerCliArgs, conf); err != nil {
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

	cobra.OnInitialize(common.InitLogging)
}
