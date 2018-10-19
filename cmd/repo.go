package cmd

import (
	"context"
	"fmt"
	_ "github.com/florinutz/gh-recruiter/github"
	github2 "github.com/florinutz/gh-recruiter/github"
	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	"time"
)

// repoCmd represents the repo command
var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "filters users who interacted with the repo by location",
	Run:   RunRepo,
	Args:  cobra.ExactArgs(2),
}

var repoConfig struct {
	location string
	csv      bool
}

func init() {
	repoCmd.Flags().StringVarP(&repoConfig.location, "location", "l", "", "location filter")
	repoCmd.Flags().BoolVarP(&repoConfig.csv, "csv", "c", false, "csv output")

	rootCmd.AddCommand(repoCmd)
}

func RunRepo(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	// get go-github client
	client := github.NewClient(oauth2.NewClient(ctx,
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: rootConfig.token}),
	))

	owner := args[0]
	repo := args[1]

	r, _, err := client.Repositories.Get(ctx, args[0], args[1])
	if err != nil {
		log.WithError(err).Fatalln("problem fetching repo")
	}
	log.WithField("repo", r).Debug("found repo info")

	fmt.Printf("Parsing repo %s\n\n", r.GetCloneURL())

	fetcher := github2.NewFetcher(client, owner, repo)

	// fetcher.ParseForks(ctx, 10, 5*time.Second, parseForksFetchResult)

	// err = fetcher.ParseContributorsStats(ctx, 10, 5*time.Second, parseContributorsStatsFetchResult)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// fetcher.ParseContributors(ctx, 10, 5*time.Second, parseContributorsFetchResult)

	fetcher.ParseStargazers(ctx, 10, 5*time.Second, parseStargazersFetchResult)

	// searchResult, _, err := client.Search.Users(ctx, "location:Berlin",
	// 	&github.SearchOptions{Sort: "forks", Order: "desc", ListOptions: github.ListOptions{PerPage: 100}})
	// if err != nil {
	// 	log.WithError(err).Errorln("problem searching users")
	// }
	// fmt.Printf("\n\nfound total %d users", searchResult.GetTotal())
}
func parseContributorsFetchResult(page int, call github2.ContributorsFetchResult) {
	err := call.Err
	if err != nil {
		if github2.IsRateLimitError(err) {
			fmt.Printf("rate limit hit while fetching page %d\n", page)
		} else {
			fmt.Printf("problem fetching page %d\n", page)
		}
	}
	for _, repo := range call.Chunk {
		fmt.Printf("%s\n", *repo.URL)
	}
}

func getRepoOwner(ctx context.Context, client *github.Client, repo *github.Repository) (*github.User, error) {
	if repo.Owner != nil {
		return repo.Owner, nil
	}

	user, _, err := client.Users.Get(ctx, repo.Owner.GetLogin())
	if err != nil {
		if github2.IsRateLimitError(err) {
			return nil, errors.Wrapf(err, "reached rate limit while fetching user %s's data", repo.Owner.GetLogin())
		} else {
			return nil, errors.Wrapf(err, "error while fetching user %s", repo.Owner.GetLogin())
		}
	}

	return user, nil
}

func parseForksFetchResult(page int, call github2.ForksFetchResult) {
	err := call.Err
	if err != nil {
		if github2.IsRateLimitError(err) {
			fmt.Printf("rate limit hit while fetching page %d\n", page)
		} else {
			fmt.Printf("problem fetching page %d\n", page)
		}
	}
	for _, repo := range call.Chunk {
		fmt.Printf("%s\n", *repo.URL)
	}
}

func parseContributorsStatsFetchResult(page int, call github2.ContributorsStatsFetchResult) {
	err := call.Err
	if err != nil {
		if github2.IsRateLimitError(err) {
			fmt.Printf("rate limit hit while fetching page %d\n", page)
		} else {
			fmt.Printf("problem fetching page %d\n", page)
		}
	}
	for _, cs := range call.Chunk {
		fmt.Printf("%s (%d)\n", cs.GetAuthor().GetLogin(), cs.GetTotal())
	}
}

func parseStargazersFetchResult(page int, call github2.StargazersFetchResult) {
	err := call.Err
	if err != nil {
		if github2.IsRateLimitError(err) {
			fmt.Printf("rate limit hit while fetching page %d\n", page)
		} else {
			fmt.Printf("problem fetching page %d\n", page)
		}
	}
	for _, sg := range call.Chunk {
		fmt.Printf("%s\n", sg.User.GetURL())
	}
}
