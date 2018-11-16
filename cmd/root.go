package cmd

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootConfig struct {
	cfgFile string
	verbose bool
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "gh-recruiter",
	Short:   "hr github needs in the cli",
	Version: "0.1",
}

// Execute runs the rootCmd
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

const (
	configName      = ".gh-recruiter"
	rootFlagVerbose = "verbose"
)

func init() {
	homeDir, err := homedir.Dir()
	if err != nil {
		log.Fatal("Could not find home dir")
	}
	rootCmd.PersistentFlags().StringVarP(&rootConfig.cfgFile, "config", "c", "",
		fmt.Sprintf("config file (default is %s/%s.toml)", homeDir, configName))

	rootCmd.PersistentFlags().BoolVarP(&rootConfig.verbose, rootFlagVerbose, "v", false,
		"Verbose?")

	viper.BindPFlag("verbose", repoCmd.Flag(rootFlagVerbose))

	cobra.OnInitialize(initConfig)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if rootConfig.cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(rootConfig.cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			log.WithError(err).Fatal()
		}

		viper.AddConfigPath(home)
		viper.AddConfigPath(".")

		viper.SetConfigName(configName)
	}

	viper.SetEnvPrefix("gr")

	if err := viper.ReadInConfig(); err != nil {
		log.WithError(err).Fatal("Can't read config")
	}
}
