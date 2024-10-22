package main

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/gleanerio/gleaner/internal/check"
	"github.com/gleanerio/gleaner/internal/config"
	"github.com/gleanerio/gleaner/pkg"
	log "github.com/sirupsen/logrus"

	"github.com/spf13/viper"

	"github.com/gleanerio/gleaner/internal/common"
	"github.com/gleanerio/gleaner/internal/objects"
)

var viperVal, sourceVal, modeVal, logVal string
var setupVal, rudeVal bool

func init() {
	common.InitLogging()
	flag.BoolVar(&setupVal, "setup", false, "Run Gleaner configuration check and exit")
	flag.StringVar(&sourceVal, "source", "", "Override config file source(s) to specify an index target")
	flag.BoolVar(&rudeVal, "rude", false, "Ignore any robots.txt crawl delays or allow / disallow statements")
	flag.StringVar(&viperVal, "cfg", "config", "Configuration file (can be YAML, JSON)")
	flag.StringVar(&modeVal, "mode", "full", "Set the mode (full | diff) to index all or just diffs")
	flag.StringVar(&logVal, "log", "warn", "The log level to output (trace | debug | info | warn | error | fatal)")
}

func main() {
	flag.Parse() // parse any command line flags...
	lvl, err := log.ParseLevel(logVal)

	if err != nil {
		log.Fatal("invalid log level:", err.Error())
	}
	log.SetLevel(lvl)

	var v1 *viper.Viper

	// Load the config file and set some defaults (config overrides)

	if isFlagPassed("cfg") {
		v1, err = config.ReadGleanerConfig(filepath.Base(viperVal), filepath.Dir(viperVal))
		if err != nil {
			log.Fatal("error when reading config:", err)
		}
	} else {
		flag.Usage()
		log.Fatal("Gleaner must be run with a config file: -cfg CONFIGFILE")
	}

	// read config file for minio info (need the bucket to check existence )
	//miniocfg := v1.GetStringMapString("minio")
	//bucketName := miniocfg["bucket"] //   get the top level bucket for all of gleaner operations from config file
	//bucketName, err := configTypes.GetBucketName(v1)

	// Remove all source EXCEPT the one in the source command lind
	if isFlagPassed("source") {
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
			log.Fatal("CAUTION:  no matching source, did your -source VALUE match a sources.name VALUE in your config file?")
		}

		configMap := v1.AllSettings()
		delete(configMap, "sources")
		v1.Set("sources", tmp)

		if rudeVal {
			v1.Set("rude", true)
		}
	} else if rudeVal {
		log.Error("--rude can only be used with --source, not globally.")
	}

	// Parse a new mode entry from command line if present
	if isFlagPassed("mode") {
		m := v1.GetStringMap("summoner")
		m["mode"] = modeVal
		v1.Set("summoner", m)
	}

	// Set up the minio connector
	mc := common.MinioConnection(v1)

	// If requested, set up the buckets
	if setupVal {
		log.Info("Setting up buckets")
		err = check.Setup(mc, v1)
		if err != nil {
			log.Fatal("Error making buckets for setup call")
		}

		log.Info("Buckets generated. Object store should be ready for runs")
		os.Exit(0)
	}

	// Validate Minio access
	err = check.PreflightChecks(mc, v1)
	if err != nil {
		log.Fatal("Preflight Check failed. Make sure the minio server is running, accessible and has been setup.", err)
	}

	//err = check.ConnCheck(mc)
	//if err != nil {vi
	//	log.Printf("Connection issue, make sure the minio server is running and accessible. %s ", err)
	//	os.Exit(1)
	//}
	//
	//// Check our bucket is ready
	//err = check.Buckets(mc, bucketName)
	//if err != nil {
	//	log.Printf("Can not find bucket. %s ", err)
	//	os.Exit(1)
	//}

	// Defer a function to be called on successful ending.  Note, if gleaner crashes, this will NOT
	// get called, do consideration must be taken in such a cases.  Some errors in such cases should
	// be sent to stdout to be captured by docker, k8s, Airflow etc in case they are being used.

	pkg.Cli(mc, v1) // move to a common call in batch.go
}

func isFlagPassed(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}
