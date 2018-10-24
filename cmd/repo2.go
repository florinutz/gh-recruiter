package cmd

import (
	"context"
	"encoding/json"
	"os"

	"github.com/shurcooL/githubv4"

	log "github.com/sirupsen/logrus"
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

type LangFragment struct {
	Id   githubv4.ID
	Name githubv4.String
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

	var crawlRepoQuery struct {
		Repository struct {
			Id              githubv4.String
			Url             githubv4.URI
			Description     githubv4.String
			ForkCount       githubv4.Int
			HomepageUrl     githubv4.URI
			NameWithOwner   githubv4.String
			PrimaryLanguage LangFragment
		} `graphql:"repository(owner:$repositoryOwner,name:$repositoryName)"`
		RateLimit struct {
			Cost      githubv4.Int
			Limit     githubv4.Int
			Remaining githubv4.Int
			ResetAt   githubv4.DateTime
		}
	}

	variables := map[string]interface{}{
		"repositoryOwner": githubv4.String(args[0]),
		"repositoryName":  githubv4.String(args[1]),
	}

	err = client.Query(ctx, &crawlRepoQuery, variables)
	if err != nil {
		log.Fatalln(err)
	}
	printJSON(crawlRepoQuery)
}

// printJSON prints v as JSON encoded with indent to stdout. It panics on any error.
func printJSON(v interface{}) {
	w := json.NewEncoder(os.Stdout)
	w.SetIndent("", "\t")
	err := w.Encode(v)
	if err != nil {
		panic(err)
	}
}
