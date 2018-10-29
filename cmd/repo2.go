package cmd

import (
	"context"
	"encoding/json"
	"errors"
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

func RunRepo2(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	oauthClient := oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: rootConfig.token}))
	githubGraphQlClient := githubv4.NewClient(oauthClient)

	var crawlRepoQuery struct {
		Repository struct {
			Id              *githubv4.String
			Url             *githubv4.URI
			Description     *githubv4.String
			HomepageUrl     *githubv4.URI
			NameWithOwner   *githubv4.String
			PrimaryLanguage LangFragment
			Forks           Forks      `graphql:"forks(first: $forksPerBatch, orderBy: {field: STARGAZERS, direction: DESC})"`
			PullRequests    PRs        `graphql:"pullRequests(first: $prsPerBatch, orderBy: {field: UPDATED_AT, direction: DESC})"`
			Releases        Releases   `graphql:"releases(first: $releasesPerBatch, orderBy: {field: CREATED_AT, direction: DESC})"`
			Stargazers      Stargazers `graphql:"stargazers(first: $stargazersPerBatch, orderBy: {field: STARRED_AT, direction: DESC})"`
		} `graphql:"repository(owner:$repositoryOwner,name:$repositoryName)"`
		RateLimit RateLimit
	}

	variables := map[string]interface{}{
		"repositoryOwner":    githubv4.String(args[0]),
		"repositoryName":     githubv4.String(args[1]),
		"forksPerBatch":      githubv4.Int(1),
		"prsPerBatch":        githubv4.Int(1),
		"prCommitsPerBatch":  githubv4.Int(1),
		"prCommentsPerBatch": githubv4.Int(1),
		"prReviewsPerBatch":  githubv4.Int(1),
		"releasesPerBatch":   githubv4.Int(1),
		"stargazersPerBatch": githubv4.Int(1),
	}

	err := githubGraphQlClient.Query(ctx, &crawlRepoQuery, variables)
	if err != nil {
		log.Fatalln(err)
	}
	printJSON(crawlRepoQuery)
}

// printJSON prints v as JSON encoded with indent to stdout. It panics on any error.
func printJSON(v interface{}) {
	if v == nil {
		log.WithError(errors.New("nil value for json")).Fatal()
	}
	w := json.NewEncoder(os.Stdout)
	w.SetIndent("", "\t")
	err := w.Encode(v)
	if err != nil {
		log.WithError(err).Fatal()
	}
}
