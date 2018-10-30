package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

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

	// variables := map[string]interface{}{
	// 	"repositoryOwner":    githubv4.String(args[0]),
	// 	"repositoryName":     githubv4.String(args[1]),
	// 	"forksPerBatch":      githubv4.Int(0),
	// 	"prsPerBatch":        githubv4.Int(0),
	// 	"prCommitsPerBatch":  githubv4.Int(0),
	// 	"prCommentsPerBatch": githubv4.Int(0),
	// 	"prReviewsPerBatch":  githubv4.Int(0),
	// 	"releasesPerBatch":   githubv4.Int(0),
	// 	"stargazersPerBatch": githubv4.Int(0),
	// }
	// err := githubGraphQlClient.Query(ctx, &repo, variables)
	// if err != nil {
	// 	log.Fatalln(err)
	// }
	//printJSON(repo)

	fetchForkers(
		func(logins []string) {
			fmt.Printf("Forkers:\n%s\n\n", strings.Join(logins, ", "))
		},
		githubGraphQlClient,
		ctx,
		args[0],
		args[1],
		100,
	)
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

type PageInfo struct {
	EndCursor   githubv4.String
	HasNextPage githubv4.Boolean
}

func fetchForkers(
	callback func(logins []string),
	client *githubv4.Client,
	ctx context.Context,
	repoOwner, repoName string,
	pageSize int) {

	data, err := getForkers(ctx, client, repoOwner, repoName, "", pageSize)
	if err != nil {
		log.WithError(err).Fatal()
	}
	callback(data)
}

type ForkNodes []struct {
	Owner struct {
		Login string
	}
}

func getPRCommenters(
	ctx context.Context,
	client *githubv4.Client,
	repoOwner string,
	repoName string,
) (results []PRComment, err error) {
	variables := map[string]interface{}{
		"repositoryOwner":    githubv4.String(repoOwner),
		"repositoryName":     githubv4.String(repoName),
		"$prsPerBatch":       githubv4.Int(100),
		"prCommentsPerBatch": githubv4.Int(100),
	}
	var q struct {
		Repository struct {
			PullRequests []struct {
				Comments struct {
					PageInfo PageInfo
					Nodes    []PRComment
				} `graphql:"comments(first: $prCommentsPerBatch)"`
			} `graphql:"pullRequests(first: $prsPerBatch, orderBy: {field: UPDATED_AT, direction: DESC})"`
		} `graphql:"repository(owner:$repositoryOwner,name:$repositoryName)"`
		RateLimit RateLimit
	}

	err = client.Query(ctx, &q, variables)
	if err != nil {
		return
	}

	for _, pr := range q.Repository.PullRequests {
		results = append(results, pr.Comments.Nodes...)
	}

	return
}

// todo filter these by location
func getForkers(
	ctx context.Context,
	client *githubv4.Client,
	repoOwner string,
	repoName string,
	after string,
	pageSize int,
) (results []string, err error) {
	var (
		nodes       ForkNodes
		hasNextPage bool
		endCursor   string
	)

	variables := map[string]interface{}{
		"repositoryOwner": githubv4.String(repoOwner),
		"repositoryName":  githubv4.String(repoName),
		"itemsPerBatch":   githubv4.Int(pageSize),
	}

	if after == "" {
		var q struct {
			Repository struct {
				Forks struct {
					PageInfo PageInfo
					Nodes    ForkNodes
				} `graphql:"forks(first: $itemsPerBatch, orderBy: {field: STARGAZERS, direction: DESC})"`
			} `graphql:"repository(owner:$repositoryOwner,name:$repositoryName)"`
			RateLimit RateLimit
		}

		err = client.Query(ctx, &q, variables)
		if err != nil {
			return
		}
		nodes = q.Repository.Forks.Nodes
		hasNextPage = bool(q.Repository.Forks.PageInfo.HasNextPage)
		endCursor = string(q.Repository.Forks.PageInfo.EndCursor)
	} else {
		var q struct {
			Repository struct {
				Forks struct {
					PageInfo PageInfo
					Nodes    ForkNodes
				} `graphql:"forks(first: $itemsPerBatch, after: $after, orderBy: {field: STARGAZERS, direction: DESC})"`
			} `graphql:"repository(owner:$repositoryOwner,name:$repositoryName)"`
			RateLimit RateLimit
		}

		variables := map[string]interface{}{
			"repositoryOwner": githubv4.String(repoOwner),
			"repositoryName":  githubv4.String(repoName),
			"itemsPerBatch":   githubv4.Int(pageSize),
			"after":           githubv4.String(after),
		}

		err = client.Query(ctx, &q, variables)
		if err != nil {
			return
		}
		nodes = q.Repository.Forks.Nodes
		hasNextPage = bool(q.Repository.Forks.PageInfo.HasNextPage)
		endCursor = string(q.Repository.Forks.PageInfo.EndCursor)
	}

	for _, owner := range nodes {
		results = append(results, owner.Owner.Login)
	}

	if hasNextPage {
		data, err := getForkers(ctx, client, repoOwner, repoName, endCursor, pageSize)
		if err != nil {
			return results, err
		}
		results = append(results, data...)
	}

	return
}
