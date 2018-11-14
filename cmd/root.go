package cmd

import (
	"fmt"
	"os"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootConfig = struct {
	token   string
	cfgFile string
}{}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gh-recruiter",
	Short: "hr github needs in the cli",
}

// Execute runs the rootCmd
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	addTokenFlag(rootCmd, &rootConfig.token, "token", "t")

	rootCmd.PersistentFlags().StringVar(&rootConfig.cfgFile, "config", "",
		"config file (default is $HOME/.recruiter.yaml)")
}

func addTokenFlag(cmd *cobra.Command, storer *string, tokenFlag, shorthand string) {
	cmd.PersistentFlags().StringVarP(storer, tokenFlag, shorthand, "",
		"Github personal access token (REQUIRED)")
	viper.BindPFlag(tokenFlag, cmd.PersistentFlags().Lookup(tokenFlag))
	cmd.MarkFlagRequired(tokenFlag)
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
			fmt.Println(err)
			os.Exit(1)
		}

		viper.AddConfigPath(home)
		viper.SetConfigName(".recruiter")
	}

	viper.SetEnvPrefix("gr")
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
	}
}
