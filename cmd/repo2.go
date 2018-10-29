package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/shurcooL/githubv4"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

// repoCmd represents the repo command
var (
	repo2Cmd = &cobra.Command{
		Use:   "repo2",
		Short: "filters users who interacted with the repo by location",
		Run:   RunRepo2,
		Args:  cobra.ExactArgs(2),
	}
	repo QueryRepo
)

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

	variables := map[string]interface{}{
		"repositoryOwner":    githubv4.String(args[0]),
		"repositoryName":     githubv4.String(args[1]),
		"forksPerBatch":      githubv4.Int(0),
		"prsPerBatch":        githubv4.Int(0),
		"prCommitsPerBatch":  githubv4.Int(0),
		"prCommentsPerBatch": githubv4.Int(0),
		"prReviewsPerBatch":  githubv4.Int(0),
		"releasesPerBatch":   githubv4.Int(0),
		"stargazersPerBatch": githubv4.Int(0),
	}

	err := githubGraphQlClient.Query(ctx, &repo, variables)
	if err != nil {
		log.Fatalln(err)
	}
	printJSON(repo)
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

type ForksFetchResult struct {
	ForkOwners []string
	Err        error
}

type ForksCallback func(page int, call ForksFetchResult)

func fetchForkers(client *githubv4.Client, ctx context.Context, repoOwner, repoName string, numberOfPages, pageSize int,
	callback ForksCallback, timeout time.Duration) {
	type QueryForForkers struct {
		Repository struct {
			Forks struct {
				Nodes []struct {
					Owner struct {
						Login string
					}
				}
			} `graphql:"forks(first: $forksPerBatch, orderBy: {field: STARGAZERS, direction: DESC})"`
		} `graphql:"repository(owner:$repositoryOwner,name:$repositoryName)"`
		RateLimit RateLimit
	}

	pageGetter := func(
		client *githubv4.Client,
		ctx context.Context,
		repoOwner, repoName string, out chan<- ForksFetchResult, page, totalPages, pageSize int) {
		var q QueryForForkers

		variables := map[string]interface{}{
			"repositoryOwner": githubv4.String(repoOwner),
			"repositoryName":  githubv4.String(repoName),
			"forksPerBatch":   githubv4.Int(pageSize),
		}

		err := client.Query(ctx, &q, variables)
		if err != nil {
			out <- ForksFetchResult{nil, err}
		}

		var logins []string
		for _, owner := range q.Repository.Forks.Nodes {
			logins = append(logins, owner.Owner.Login)
		}

		out <- ForksFetchResult{logins, err}

		if totalPages != 0 && totalPages == len(out) {
			close(out)
		}
	}

	resultsChan := make(chan ForksFetchResult)

	if numberOfPages > 1 {
		for page := 1; page <= numberOfPages; page++ {
			go pageGetter(client, ctx, repoOwner, repoName, resultsChan, page, numberOfPages, pageSize)
		}
	}

	for page := 0; page <= numberOfPages; page++ {
		select {
		case data := <-resultsChan:
			callback(page, data)
		case <-time.After(timeout):
			fmt.Println("timeout")
		}
	}
}
