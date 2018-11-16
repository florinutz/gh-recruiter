package cmd

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/florinutz/gh-recruiter/cache"
	"github.com/florinutz/gh-recruiter/fetch"
	"github.com/shurcooL/githubv4"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

// repoCmd represents the repo command
var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "filters users who interacted with the repo by location",
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindEnv("token")
		RepoCmdConfig.Token = viper.GetString("token")
		viper.UnmarshalKey("repo", &RepoCmdConfig.Repos)
	},
	Run:  runRepo,
	Args: cobra.ExactArgs(2),
}

// RepoSettings represents the settings for individual repos
type RepoSettings struct {
	Name        string `toml:"name" comment:"<owner>/<repoName>"`
	Csv         string `toml:"csv_output" commented:"true" comment:"if this is present, csv will pe outputted at the desired path"`
	Verbose     bool   `toml:"verbose" commented:"true" comment:"too much output will be shown, but some might enjoy this"`
	WithForkers bool   `toml:"forkers" commented:"true" comment:"analyze forkers"`
	WithPRs     bool   `toml:"prs" commented:"true" comment:"analyze PRs"`
}

type RepoConfig struct {
	Token       string         `toml:"token" comment:"github token. Supplying it as the GR_TOKEN env var will take precedence over this config file"`
	Csv         string         `toml:"csv_output" commented:"true" comment:"root setting for csv output. Can be overwritten at repo level"`
	Verbose     bool           `toml:"verbose" commented:"true" comment:"show more output"`
	WithForkers bool           `toml:"forkers" comment:"parse forkers"`
	WithPRs     bool           `toml:"prs" commented:"true" comment:"parse prs"`
	Repos       []RepoSettings `toml:"repos" comment:"each repository can overwrite the base settings"`
}

// RepoCmdConfig covers all config options for this command
var RepoCmdConfig RepoConfig

const (
	cacheBucketName = "gh-recruiter"
	repoFlagOutput  = "output"
	repoFlagForkers = "forkers"
	repoFlagPrs     = "prs"
)

func init() {
	repoCmd.Flags().StringVarP(&RepoCmdConfig.Csv, repoFlagOutput, "o", "",
		"Csv output file")

	repoCmd.Flags().BoolVarP(&RepoCmdConfig.WithForkers, repoFlagForkers, "f", false,
		"fetch forkers?")

	repoCmd.Flags().BoolVarP(&RepoCmdConfig.WithPRs, repoFlagPrs, "p", false,
		"fetch users involved in prs?")

	viper.BindPFlag("csv_output", repoCmd.Flag(repoFlagOutput))
	viper.BindPFlag("forkers", repoCmd.Flag(repoFlagForkers))
	viper.BindPFlag("prs", repoCmd.Flag(repoFlagPrs))

	rootCmd.AddCommand(repoCmd)
}

func runRepo(cmd *cobra.Command, args []string) {
	fmt.Printf("%q\n", RepoCmdConfig.Repos)
	return
	c, err := cache.NewCache(cacheBucketName, 168*time.Hour)
	if err != nil {
		log.Warnf("Running with no cache: %s\n", err)
	}
	ctx := context.Background()
	oauthClient := oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: RepoCmdConfig.Token}))
	ghClient := githubv4.NewClient(oauthClient)

	g := fetch.GithubFetcher{Client: ghClient, Cache: c}

	if RepoCmdConfig.WithForkers {
		logins, err := g.GetForkers(ctx, args[0], args[1], (*githubv4.String)(nil), 100)
		if err != nil {
			log.WithError(err).Fatal()
		}

		var writer *csv.Writer
		if RepoCmdConfig.Csv != "" {
			path := fmt.Sprintf("%s_%s-%s_forkers.Csv", RepoCmdConfig.Csv, args[0], args[1])
			writer = MustInitCsv(path, true)
		}
		g.GetUsersByLogins(ctx, logins, writer, userFetchedCallback)
	}

	if RepoCmdConfig.WithPRs {
		prs, err := g.GetPRs(ctx, args[0], args[1], (*githubv4.String)(nil), 0)
		if err != nil {
			log.WithError(err).Fatal()
		}

		// need to harvest these
		var (
			commenterLogins []string
			reviewerLogins  []string
		)
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
			g.GetUsersByLogins(ctx, commenterLogins, writer, userFetchedCallback)
		}

		if len(reviewerLogins) > 0 {
			var writer *csv.Writer
			if RepoCmdConfig.Csv != "" {
				path := fmt.Sprintf("%s_%s-%s_pr_reviewers.Csv", RepoCmdConfig.Csv, args[0], args[1])
				writer = MustInitCsv(path, true)
			}
			g.GetUsersByLogins(ctx, reviewerLogins, writer, userFetchedCallback)
		}
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
