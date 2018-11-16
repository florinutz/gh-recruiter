package cmd

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml"

	"github.com/pkg/errors"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// repoCmd represents the repo command
var configCmd = &cobra.Command{
	Use:   "gen-config",
	Short: "generate config",
	Run:   runConfig,
	Args:  cobra.ExactArgs(1),
}

func init() {
	rootCmd.AddCommand(configCmd)
}

func runConfig(cmd *cobra.Command, args []string) {
	incoming := args[0]
	if !filepath.IsAbs(incoming) {
		var err error
		if incoming, err = filepath.Abs(incoming); err != nil {
			log.WithError(err).Fatal()
		}
	}
	if !pathIsValid(incoming) {
		log.WithError(errors.New("invalid path")).Fatal()
	}

	f, err := os.Create(incoming)
	if err != nil {
		log.WithError(err).Fatal()
	}
	defer f.Close()

	conf := RepoConfig{
		Token:       "your_github_token_here",
		Verbose:     false,
		WithForkers: true,
		Csv:         "/tmp/testing_this_",
		Repos: []RepoSettings{
			{
				Name:    "hashicorp/hcl",
				Verbose: true,
				Csv:     "/tmp/gh-hcl",
			},
			{
				Name:        "openzipkin/zipkin-go",
				WithForkers: true,
			},
		},
	}
	if err := toml.NewEncoder(f).Encode(conf); err != nil {
		log.WithError(err).Fatal()
	}
}

func pathIsValid(fp string) bool {
	// Check if file already exists
	if _, err := os.Stat(fp); err == nil {
		return true
	}
	// Attempt to create it
	var d []byte
	if err := ioutil.WriteFile(fp, d, 0644); err == nil {
		os.Remove(fp) // And delete it
		return true
	}
	return false
}
