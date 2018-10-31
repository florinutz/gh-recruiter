package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
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

	// forkerLogins, err := getForkers(ctx, githubGraphQlClient, args[0], args[1], "", 100)
	// if err != nil {
	// 	log.WithError(err).Fatal()
	// }
	logins := []string{"florinutz", "nocive"}
	out := getUsersByLogins(logins, ctx, githubGraphQlClient, 5*time.Second)

	for i := 0; i < len(logins); i++ {
		select {
		case userFetch := <-out:
			fmt.Printf("%s: %s\n", userFetch.Login, userFetch.User.Bio)
		case <-time.After(10 * time.Second):
			fmt.Println("timeout")
		}
	}

	// prs, err := getPRs(ctx, githubGraphQlClient, args[0], args[1], "")
	// if err != nil {
	// 	log.WithError(err).Fatal()
	// }
	//
	// for _, pr := range prs {
	// 	fmt.Printf("\n\nPR %s (%s):\n", pr.Title, pr.Url)
	// 	commentsCount := len(pr.Comments.Nodes)
	// 	if commentsCount > 0 {
	// 		fmt.Printf("\n%d comments:\n", commentsCount)
	// 		for _, comment := range pr.Comments.Nodes {
	// 			fmt.Printf("%s (%s):\n", comment.Author.Login, comment.Url.String())
	// 		}
	// 	}
	// 	reviewsCount := len(pr.Reviews.Nodes)
	// 	if reviewsCount > 0 {
	// 		fmt.Printf("\n%d reviews:\n", reviewsCount)
	// 		for _, review := range pr.Reviews.Nodes {
	// 			fmt.Printf("%s (%s):\n", review.Author.Login, review.Url.String())
	// 		}
	// 	}
	// 	commitsCount := len(pr.Commits.Nodes)
	// 	if commitsCount > 0 {
	// 		fmt.Printf("\n%d commits:\n", commitsCount)
	// 		for _, commit := range pr.Commits.Nodes {
	// 			fmt.Printf("%s (%d additions, %d deletions, url %s):\n",
	// 				commit.Commit.Author.User.Id, commit.Commit.Additions, commit.Commit.Url)
	// 		}
	// 	}
	// }
}

func getUsersByLogins(logins []string, ctx context.Context, client *githubv4.Client, timeout time.Duration) chan UserFetchResult {
	out := make(chan UserFetchResult)
	sent := 0
	for _, login := range logins {
		duration := time.Duration(rand.Intn(len(logins))) * time.Second // about 1 per second but randomly
		go func(login string, wait time.Duration) {
			time.Sleep(wait)

			user, err := getUser(login, ctx, client)

			out <- UserFetchResult{login, user, err}
			sent++
			if sent == len(logins) {
				close(out)
			}
		}(login, duration)
	}
	return out
}

type UserFetchResult struct {
	Login string
	User  UserFragment
	Err   error
}

func getUser(login string, ctx context.Context, client *githubv4.Client) (UserFragment, error) {
	var q struct {
		User      UserFragment `graphql:"user(login:$login)"`
		RateLimit RateLimit
	}
	err := client.Query(ctx, &q, map[string]interface{}{"login": githubv4.String(login), "maxOrgs": githubv4.Int(2)})
	if err != nil {
		return UserFragment{}, err
	}

	return q.User, nil
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

type ForkNodes []struct {
	Owner struct {
		Login string
	}
}

type PRWithData struct {
	Url      githubv4.URI
	Title    githubv4.String
	Comments struct {
		Nodes []PRComment
	} `graphql:"comments(first: $prItemsPerBatch)"`
	Reviews struct {
		Nodes []PRReview
	} `graphql:"comments(first: $prItemsPerBatch)"`
	Commits struct {
		Nodes []PRCommit
	} `graphql:"commits(first: $prCommitsPerBatch)"`
}

func getPRs(
	ctx context.Context,
	client *githubv4.Client,
	repoOwner string,
	repoName string,
	after string,
) (results []PRWithData, err error) {
	var (
		nodes       []PRWithData
		hasNextPage bool
		endCursor   string
	)

	variables := map[string]interface{}{
		"repositoryOwner":   githubv4.String(repoOwner),
		"repositoryName":    githubv4.String(repoName),
		"prsPerBatch":       githubv4.Int(100),
		"prItemsPerBatch":   githubv4.Int(100),
		"prCommitsPerBatch": githubv4.Int(5), // a safe value so that we don't request too much data
	}

	if after != "" {
		var q struct {
			Repository struct {
				PullRequests struct {
					PageInfo PageInfo
					Nodes    []PRWithData
				} `graphql:"pullRequests(after: $after, first: $prsPerBatch, orderBy: {field: UPDATED_AT, direction: DESC})"`
			} `graphql:"repository(owner:$repositoryOwner,name:$repositoryName)"`
			RateLimit RateLimit
		}
		variables["after"] = githubv4.String(after)

		err = client.Query(ctx, &q, variables)
		if err != nil {
			return
		}

		nodes = q.Repository.PullRequests.Nodes
		hasNextPage = bool(q.Repository.PullRequests.PageInfo.HasNextPage)
		endCursor = string(q.Repository.PullRequests.PageInfo.EndCursor)
	} else {
		var q struct {
			Repository struct {
				PullRequests struct {
					PageInfo PageInfo
					Nodes    []PRWithData
				} `graphql:"pullRequests(first: $prsPerBatch, orderBy: {field: UPDATED_AT, direction: DESC})"`
			} `graphql:"repository(owner:$repositoryOwner,name:$repositoryName)"`
			RateLimit RateLimit
		}

		err = client.Query(ctx, &q, variables)
		if err != nil {
			return
		}

		nodes = q.Repository.PullRequests.Nodes
		hasNextPage = bool(q.Repository.PullRequests.PageInfo.HasNextPage)
		endCursor = string(q.Repository.PullRequests.PageInfo.EndCursor)
	}

	results = append(results, nodes...)

	if hasNextPage {
		data, err := getPRs(ctx, client, repoOwner, repoName, endCursor)
		if err != nil {
			return results, err
		}
		results = append(results, data...)
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
