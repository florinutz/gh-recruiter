package cmd

import (
	"context"
	"log"

	"github.com/shurcooL/githubv4"

	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

// repoCmd represents the repo command
var repo2Cmd = &cobra.Command{
	Use:   "repo2",
	Short: "filters users who interacted with the repo by location",
	Run:   RunRepo2,
	Args:  cobra.ExactArgs(2),
}

var repo2Config struct {
	location string
	csv      bool
}

func init() {
	repo2Cmd.Flags().StringVarP(&repo2Config.location, "location", "l", "", "location filter")
	repo2Cmd.Flags().BoolVarP(&repo2Config.csv, "csv", "c", false, "csv output")

	rootCmd.AddCommand(repo2Cmd)
}

func RunRepo2(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	client := githubv4.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: rootConfig.token})))

	var query struct {
		Viewer struct {
			Login     githubv4.String
			CreatedAt githubv4.DateTime
		}
	}

	err := client.Query(ctx, &query, nil)
	if err != nil {
		log.Fatalln(err)
	}
}
