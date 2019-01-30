package cmd

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"

	"github.com/florinutz/gh-recruiter/cache"
	"github.com/florinutz/gh-recruiter/fetch"
	"github.com/shurcooL/githubv4"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

const (
	cacheBucketName = "gh-recruiter"

	repoFlagCsvOutput = "output"
	repoFlagForkers   = "forkers"
	repoFlagPrs       = "prs"
	repoFlagRepos     = "repo"
)

type RepoSettings struct {
	Tokens  []string `toml:"tokens" commented:"false" comment:"Pool of github token to be used randomly. \n Supplying tokens via the GR_TOKEN env var will take precedence over this."`
	Csv     string   `toml:"csv" commented:"true" comment:"if this is present, csv will pe outputted at the desired path" omitempty:"true"`
	Verbose bool     `toml:"verbose" comment:"too much output will be shown, but some might enjoy this" omitempty:"true"`
	Forkers bool     `toml:"forkers" comment:"analyze forkers" omitempty:"true"`
	PRs     bool     `toml:"prs" commented:"true" comment:"analyze PRs" omitempty:"true"`
}

// repo represents the settings for individual repos
type repo struct {
	Owner string `toml:"owner" comment:"repo owner" omitempty:"false"`
	Name  string `toml:"name" comment:"repo owner" omitempty:"false"`
	RepoSettings
}

// RepoConfig represents configs for this command
type RepoConfig struct {
	RepoSettings `toml:"global" comment:"global settings that will be overridden by individual repo settings"`
	Repos        []*repo `toml:"repos" comment:"each repository can overwrite the base settings"`
}

// RepoCmdConfig covers all config options for this command
var (
	RepoCmdConfig RepoConfig
	Fetcher       fetch.GithubFetcher
)

// repoCmd represents the repo command
var repoCmd = &cobra.Command{
	Use:    "repo",
	Short:  "filters users who interacted with the repo by location",
	PreRun: preRunRepo,
	Run:    runRepo,
	Args:   cobra.ExactArgs(2),
}

func init() {
	if veep == nil {
		veep = viper.New()
	}

	repoCmd.Flags().StringVarP(&RepoCmdConfig.Csv, repoFlagCsvOutput, "o", "",
		"Csv output file")
	repoCmd.Flags().BoolVarP(&RepoCmdConfig.Forkers, repoFlagForkers, "f", false,
		"fetch forkers?")
	repoCmd.Flags().BoolVarP(&RepoCmdConfig.PRs, repoFlagPrs, "p", false,
		"fetch users involved in prs?")

	veep.BindEnv("token")

	if err := veep.BindPFlag("csv", repoCmd.Flag(repoFlagCsvOutput)); err != nil {
		log.WithError(err).Fatal("config binding error")
	}
	if err := veep.BindPFlag("forkers", repoCmd.Flag(repoFlagForkers)); err != nil {
		log.WithError(err).Fatal("config binding error")
	}
	if err := veep.BindPFlag("prs", repoCmd.Flag(repoFlagPrs)); err != nil {
		log.WithError(err).Fatal("config binding error")
	}

	rootCmd.AddCommand(repoCmd)
}

func preRunRepo(cmd *cobra.Command, args []string) {
	var err error
	s := veep.AllSettings()
	if err = veep.Unmarshal(&RepoCmdConfig); err != nil {
		log.WithError(err).WithField("cca", s).Fatal("couldn't parse config")
	}
	log.WithField("config", RepoCmdConfig).Debug("fetched config")

	ctx := context.Background()

	token := RepoCmdConfig.Tokens[0]
	ghClient := githubv4.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token})))
	log.Debugf("Github access token: %s\n", token)

	var c *cache.Cache
	if c, err = cache.NewCache(cacheBucketName, 168*time.Hour); err != nil {
		log.WithError(err).Warn("running with no cache")
	} else {
		log.WithField("cache", c).Debug("got cache")
	}

	Fetcher = fetch.GithubFetcher{Client: ghClient, Cache: c}
}

func runRepo(cmd *cobra.Command, args []string) {
	//ctx := context.Background()
	// todo merge individual repo settings over the root ones
	if RepoCmdConfig.Forkers {
		// DoForkers(ctx, args)
	}
	if RepoCmdConfig.PRs {
		// DoPRs(ctx, args)
	}
}

func (r *repo) DoForkers(ctx context.Context, owner, repo string) {
	logins, err := Fetcher.GetForkers(ctx, owner, repo, (*githubv4.String)(nil), 100)
	if err != nil {
		log.WithError(err).Fatal()
	}
	var writer *csv.Writer
	if RepoCmdConfig.Csv != "" {
		path := fmt.Sprintf("%s_%s-%s_forkers.csv", RepoCmdConfig.Csv, owner, repo)
		writer = MustInitCsv(path, true)
	}
	Fetcher.GetUsersByLogins(ctx, logins, writer, userFetchedCallback)
}

func (r *repo) DoPRs(ctx context.Context, args []string) {
	var commenterLogins, reviewerLogins []string

	prs, err := Fetcher.GetPRs(ctx, args[0], args[1], (*githubv4.String)(nil), 0)
	if err != nil {
		log.WithError(err).Fatal()
	}

	for _, pr := range prs {
		fmt.Printf("\n\nPR %s (%s):\n", pr.Title, pr.URL)

		commentsCount := len(pr.Comments.Nodes)
		if commentsCount > 0 {
			fmt.Printf("\n%d comments:\n", commentsCount)
			for _, comment := range pr.Comments.Nodes {
				fmt.Printf("%s (%s):\n", comment.Author.Login, comment.URL.String())
				commenterLogins = append(commenterLogins, string(comment.Author.Login))
			}
		}

		reviewsCount := len(pr.Reviews.Nodes)
		if reviewsCount > 0 {
			fmt.Printf("\n%d reviews:\n", reviewsCount)
			for _, review := range pr.Reviews.Nodes {
				fmt.Printf("%s (%s):\n", review.Author.Login, review.URL.String())
				reviewerLogins = append(reviewerLogins, string(review.Author.Login))
			}
		}

		commitsCount := len(pr.Commits.Nodes)
		if commitsCount > 0 {
			fmt.Printf("\n%d commits:\n", commitsCount)

			var writer *csv.Writer
			if RepoCmdConfig.Csv != "" {
				path := fmt.Sprintf("%s_%s-%s_pr_commits.Csv", RepoCmdConfig.Csv, args[0], args[1])
				writer = MustInitCsv(path, true)
			}

			for _, commit := range pr.Commits.Nodes {
				fmt.Printf("%s (%d additions, %d deletions, url %s):\n",
					commit.Commit.Author.User.ID,
					commit.Commit.Additions,
					commit.Commit.Deletions,
					commit.Commit.URL,
				)
				if writer != nil {
					writer.Write(commit.Commit.Author.User.FormatForCsv())
				}
			}
		}
	}
	if len(commenterLogins) > 0 {
		var writer *csv.Writer
		if RepoCmdConfig.Csv != "" {
			path := fmt.Sprintf("%s_%s-%s_pr_commenters.Csv", RepoCmdConfig.Csv, args[0], args[1])
			writer = MustInitCsv(path, true)
		}
		Fetcher.GetUsersByLogins(ctx, commenterLogins, writer, userFetchedCallback)
	}
	if len(reviewerLogins) > 0 {
		var writer *csv.Writer
		if RepoCmdConfig.Csv != "" {
			path := fmt.Sprintf("%s_%s-%s_pr_reviewers.Csv", RepoCmdConfig.Csv, args[0], args[1])
			writer = MustInitCsv(path, true)
		}
		Fetcher.GetUsersByLogins(ctx, reviewerLogins, writer, userFetchedCallback)
	}
}

func userFetchedCallback(ctx context.Context, fetched fetch.UserFetchResult, csvWriter *csv.Writer) {
	if fetched.Err != nil {
		log.WithError(fetched.Err).Warn()
		return
	}

	if isLocationInteresting(string(fetched.User.Location)) {
		fmt.Printf("%q\n", fetched.User.FormatForCsv())
		if csvWriter != nil {
			csvWriter.Write(fetched.User.FormatForCsv())
			csvWriter.Flush()
		}
	} else if rootConfig.verbose {
		fmt.Fprintf(os.Stderr, "%s's \"%s\" location was not interesting\n",
			fetched.Login, fetched.User.Location)
	}
}

func isLocationInteresting(location string) bool {
	var interestingLocationKeywords = []string{"germany", "deutschland", "poland", "berlin", "hamburg", "hamburg",
		"hanover", "leipzig", "dresden"}
	lowerLocation := strings.ToLower(location)
	for _, loc := range interestingLocationKeywords {
		if strings.Contains(lowerLocation, strings.ToLower(loc)) {
			return true
		}
	}

	return false
}

// MustInitCsv makes sure we have a csv to write to
func MustInitCsv(csvPath string, writeHeader bool) *csv.Writer {
	var (
		csvFile *os.File
		err     error
	)
	csvFile, err = os.OpenFile(csvPath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		log.WithError(err).Fatal()
	}
	w := csv.NewWriter(csvFile)

	if writeHeader {
		w.Write([]string{
			"Login",
			"Location",
			"Email",
			"Name",
			"Company",
			"Bio",
			"Registered",
			"Followers",
			"Following",
			"Organisations",
			"Hireable",
		})
		w.Flush()
	}
	if err := w.Error(); err != nil {
		log.WithError(err).Fatal()
	}

	return w
}
