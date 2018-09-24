package cmd

import (
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	"github.com/google/go-github/github"
	"context"
	log "github.com/sirupsen/logrus"
	"strings"
	"fmt"
)

// repoCmd represents the repo command
var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: "6406ad6ccc97fec5abc99388459e50beb8d7a640"},
		)
		tc := oauth2.NewClient(ctx, ts)

		// get go-github client
		client := github.NewClient(tc)

		owner := args[0]
		repo := args[1]

		r, _, err := client.Repositories.Get(ctx, args[0], args[1])
		if err != nil {
			log.WithError(err).Errorln("problem fetching repo")
		}
		log.WithField("repo", r).Debug("found repo info")

		fmt.Printf("Parsing repo %s\n\n", r.GetCloneURL())

		// contributorsStats, _, err := client.Repositories.ListContributorsStats(ctx, args[0], args[1])
		// for _, cs := range contributorsStats {
		// 	fmt.Printf("%s (%d), ", cs.GetAuthor().GetLogin(), cs.GetTotal())
		// }

		var logins []string
		parseContributors(ctx, client, owner, repo, func(contributor *github.Contributor) {
			logins = append(logins, contributor.GetLogin())
		})
		fmt.Printf("Contributors: \n\n%s\n\n", strings.Join(logins, ", "))

		var forks []string
		parseForks(ctx, client, owner, repo, func(repo *github.Repository) {
			forks = append(forks, repo.GetFullName())
		})
		fmt.Printf("Forks: \n\n%s\n\n", strings.Join(forks, ", "))

		var stargazers []string
		parseStargazers(ctx, client, owner, repo, func(stargazer *github.Stargazer) {
			stargazers = append(forks, stargazer.GetUser().GetLogin())
		})
		fmt.Printf("Stargazers: \n\n%s\n\n", strings.Join(stargazers, ", "))

		// searchResult, _, err := client.Search.Users(ctx, "location:Berlin",
		// 	&github.SearchOptions{Sort: "forks", Order: "desc", ListOptions: github.ListOptions{PerPage: 100}})
		// if err != nil {
		// 	log.WithError(err).Errorln("problem searching users")
		// }
		// fmt.Printf("\n\nfound total %d users", searchResult.GetTotal())
	},
	Args: cobra.ExactArgs(2),
}

func parseContributors(
	ctx context.Context,
	client *github.Client,
	owner string,
	repo string,
	callback func(contributor *github.Contributor)) {

	page := 0

	for true {
		page++

		contributors, response, err := client.Repositories.ListContributors(ctx, owner, repo,
			&github.ListContributorsOptions{Anon: "false", ListOptions: github.ListOptions{Page: page, PerPage: 200}})

		if err != nil {
			log.WithError(err).Errorln("problem fetching contributors")
			continue
		}

		for _, c := range contributors {
			callback(c)
		}

		if page > response.LastPage {
			break
		}
	}
}

func parseForks(
	ctx context.Context,
	client *github.Client,
	owner string,
	repo string,
	callback func(repo *github.Repository)) {

	page := 0

	for true {
		page++

		repos, response, err := client.Repositories.ListForks(ctx, owner, repo,
			&github.RepositoryListForksOptions{ListOptions: github.ListOptions{Page: page, PerPage: 200}})

		if err != nil {
			log.WithError(err).Errorln("problem fetching forks")
			continue
		}

		for _, r := range repos {
			callback(r)
		}

		if page > response.LastPage {
			break
		}
	}
}
func parseStargazers(
	ctx context.Context,
	client *github.Client,
	owner string,
	repo string,
	callback func(stargazer *github.Stargazer)) {

	page := 0

	for true {
		page++

		stargazers, response, err := client.Activity.ListStargazers(ctx, owner, repo,
			&github.ListOptions{Page: page, PerPage: 200})

		if err != nil {
			log.WithError(err).Errorln("problem fetching stargazers")
			continue
		}

		for _, s := range stargazers {
			callback(s)
		}

		if page > response.LastPage {
			break
		}
	}
}

func init() {
	rootCmd.AddCommand(repoCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// repoCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// repoCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
