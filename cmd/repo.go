package cmd

import (
	"context"
	"fmt"
	. "github.com/florinutz/gh-recruiter/github"
	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	"os"
	"sync"
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

	r, _, err := client.Repositories.Get(ctx, args[0], args[1])
	if err != nil {
		log.WithError(err).Fatalln("problem fetching repo")
	}
	log.WithField("repo", r).Debug("found repo info")
	fmt.Printf("Parsing repo %s\n\n", r.GetCloneURL())

	cache := NewS3Cache(
		"https://s3-eu-central-1.amazonaws.com/gh-recruiter",
		os.Getenv("AWS_ACCESS_KEY_ID"),
		os.Getenv("AWS_SECRET_KEY"),
	)
	fetcher := NewFetcher(client, args[0], args[1], cache)

	var wg sync.WaitGroup
	funcs := fetcher.GetFuncs(ctx,
		parseContributorsFetchResult,
		parseContributorsStatsFetchResult,
		parseForksFetchResult,
		parseStargazersFetchResult)

	wg.Add(len(funcs))
	for _, f := range funcs {
		go func() {
			defer wg.Done()
			f()
		}()
	}
	wg.Wait()

	// searchResult, _, err := client.Search.Users(ctx, "location:Berlin",
	// 	&github.SearchOptions{Sort: "forks", Order: "desc", ListOptions: github.ListOptions{PerPage: 100}})
	// if err != nil {
	// 	log.WithError(err).Errorln("problem searching users")
	// }
	// fmt.Printf("\n\nfound total %d users", searchResult.GetTotal())
}

func fillRepoOwner(ctx context.Context, client *github.Client, repo *github.Repository) error {
	user, _, err := client.Users.Get(ctx, repo.Owner.GetLogin())
	if err != nil {
		if IsRateLimitError(err) {
			return errors.Wrapf(err, "reached rate limit while fetching user %s's data", repo.Owner.GetLogin())
		} else {
			return errors.Wrapf(err, "error while fetching user %s", repo.Owner.GetLogin())
		}
	}
	repo.Owner = user

	return nil
}

func parseContributorsFetchResult(page int, call ContributorsFetchResult) {
	err := call.Err
	if err != nil {
		if IsRateLimitError(err) {
			fmt.Printf("rate limit hit while fetching page %d\n", page)
		} else {
			fmt.Printf("problem fetching page %d\n", page)
		}
	}
	for _, repo := range call.Chunk {
		fmt.Printf("%s\n", *repo.URL)
	}
}

func parseForksFetchResult(page int, call ForksFetchResult) {
	err := call.Err
	if err != nil {
		if IsRateLimitError(err) {
			fmt.Printf("rate limit hit while fetching page %d\n", page)
		} else {
			fmt.Printf("problem fetching page %d\n", page)
		}
	}
	for _, repo := range call.Chunk {
		fmt.Printf("%s\n", *repo.URL)
	}
}

func parseContributorsStatsFetchResult(page int, call ContributorsStatsFetchResult) {
	err := call.Err
	if err != nil {
		if IsRateLimitError(err) {
			fmt.Printf("rate limit hit while fetching page %d\n", page)
		} else {
			fmt.Printf("problem fetching page %d\n", page)
		}
	}
	for _, cs := range call.Chunk {
		fmt.Printf("%s (%d)\n", cs.GetAuthor().GetLogin(), cs.GetTotal())
	}
}

func parseStargazersFetchResult(page int, call StargazersFetchResult) {
	err := call.Err
	if err != nil {
		if IsRateLimitError(err) {
			fmt.Printf("rate limit hit while fetching page %d\n", page)
		} else {
			fmt.Printf("problem fetching page %d\n", page)
		}
	}
	for _, sg := range call.Chunk {
		fmt.Printf("%s\n", sg.User.GetURL())
	}
}
