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
	"strings"
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
}

func init() {
	repoCmd.Flags().StringVarP(&repoConfig.location, "location", "l", "",
		"location filter")

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

	// contributorsStats, _, err := client.Repositories.ListContributorsStats(ctx, args[0], args[1])
	// for _, cs := range contributorsStats {
	// 	fmt.Printf("%s (%d), ", cs.GetAuthor().GetLogin(), cs.GetTotal())
	// }

	// var logins []string
	// parseContributors(ctx, client, owner, repo, func(contributor *github.Contributor) {
	// 	logins = append(logins, contributor.GetLogin())
	// })
	// fmt.Printf("Contributors: \n\n%s\n\n", strings.Join(logins, ", "))

	err = fetcher.ParseForks(ctx, func(reposChunk []*github.Repository) error {
		var strPieces []string

		for _, repo := range reposChunk {
			err = fillRepoOwner(ctx, client, repo)
			if err != nil {
				log.WithError(err).Fatal("error while fetching repo user")
			}

			var piece string

			if repoConfig.location != "" {
				if strings.Contains(strings.ToLower(repo.GetOwner().GetLocation()),
					strings.ToLower(repoConfig.location)) {
					piece = fmt.Sprintf("%s (%s)", repo.GetCloneURL(), repo.GetOwner().GetLocation())
				}
			} else {
				if repo.GetOwner().GetLocation() == "" {
					piece = fmt.Sprintf("%s", repo.GetCloneURL())
				} else {
					piece = fmt.Sprintf("%s (%s)", repo.GetCloneURL(), repo.GetOwner().GetLocation())
				}
				strPieces = append(strPieces)
			}

			if piece != "" {
				strPieces = append(strPieces, piece)
			}
		}

		fmt.Print(strings.Join(strPieces, ", ") + ", ")

		return nil
	})

	if err != nil {
		log.WithError(err).Fatal("error while parsing forks")
	}

	// var stargazers []string
	// parseStargazers(ctx, client, owner, repo, func(stargazer *github.Stargazer) {
	// 	stargazers = append(forks, stargazer.GetUser().GetLogin())
	// })
	// fmt.Printf("Stargazers: \n\n%s\n\n", strings.Join(stargazers, ", "))

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
		if github2.IsRateLimitError(err) {
			return errors.Wrapf(err, "reached rate limit while fetching user %s's data", repo.Owner.GetLogin())
		} else {
			return errors.Wrapf(err, "error while fetching user %s", repo.Owner.GetLogin())
		}
	}
	repo.Owner = user

	return nil
}
