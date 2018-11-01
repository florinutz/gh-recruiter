package cmd

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strings"
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
	CsvWriterPRReviewers *csv.Writer
	CsvWriterPRCommiters *csv.Writer
)

var repo2Config struct {
	csv     string
	verbose bool

	withForkers bool
	withPRs     bool
}

func init() {
	repo2Cmd.Flags().StringVarP(&repo2Config.csv, "output", "o", "", "csv output file")
	repo2Cmd.Flags().BoolVarP(&repo2Config.verbose, "verbose", "v", false, "verbose?")
	repo2Cmd.Flags().BoolVarP(&repo2Config.withForkers, "forkers", "f", false, "fetch forkers?")
	repo2Cmd.Flags().BoolVarP(&repo2Config.withPRs, "prs", "p", false,
		"fetch users involved in PRs?")

	rootCmd.AddCommand(repo2Cmd)
}

func RunRepo2(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	oauthClient := oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: rootConfig.token}))
	ghClient := githubv4.NewClient(oauthClient)

	if repo2Config.withForkers {
		logins, err := getForkers(ctx, ghClient, args[0], args[1], "", 100)
		if err != nil {
			log.WithError(err).Fatal()
		}

		var writer *csv.Writer
		if repo2Config.csv != "" {
			path := fmt.Sprintf("%s_%s-%s_forkers.csv", repo2Config.csv, args[0], args[1])
			writer = MustInitCsv(path, true)
		}
		GetUsersByLogins(logins, ctx, ghClient, writer, userFetchedCallback)
	}

	if repo2Config.withPRs {
		prs, err := getPRs(ctx, ghClient, args[0], args[1], "")
		if err != nil {
			log.WithError(err).Fatal()
		}

		// need to harvest these
		var (
			commenterLogins []string
			reviewerLogins  []string
		)
		for _, pr := range prs {
			fmt.Printf("\n\nPR %s (%s):\n", pr.Title, pr.Url)

			commentsCount := len(pr.Comments.Nodes)
			if commentsCount > 0 {
				fmt.Printf("\n%d comments:\n", commentsCount)
				for _, comment := range pr.Comments.Nodes {
					fmt.Printf("%s (%s):\n", comment.Author.Login, comment.Url.String())
					commenterLogins = append(commenterLogins, string(comment.Author.Login))
				}
			}

			reviewsCount := len(pr.Reviews.Nodes)
			if reviewsCount > 0 {
				fmt.Printf("\n%d reviews:\n", reviewsCount)
				for _, review := range pr.Reviews.Nodes {
					fmt.Printf("%s (%s):\n", review.Author.Login, review.Url.String())
					reviewerLogins = append(reviewerLogins, string(review.Author.Login))
				}
			}

			commitsCount := len(pr.Commits.Nodes)
			if commitsCount > 0 {
				fmt.Printf("\n%d commits:\n", commitsCount)

				var writer *csv.Writer
				if repo2Config.csv != "" {
					path := fmt.Sprintf("%s_%s-%s_pr_commits.csv", repo2Config.csv, args[0], args[1])
					writer = MustInitCsv(path, true)
				}

				for _, commit := range pr.Commits.Nodes {
					fmt.Printf("%s (%d additions, %d deletions, url %s):\n",
						commit.Commit.Author.User.Id,
						commit.Commit.Additions,
						commit.Commit.Deletions,
						commit.Commit.Url,
					)
					if writer != nil {
						writer.Write(commit.Commit.Author.User.FormatForCsv())
					}
				}
			}
		}

		if len(commenterLogins) > 0 {
			var writer *csv.Writer
			if repo2Config.csv != "" {
				path := fmt.Sprintf("%s_%s-%s_pr_commenters.csv", repo2Config.csv, args[0], args[1])
				writer = MustInitCsv(path, true)
			}
			GetUsersByLogins(commenterLogins, ctx, ghClient, writer, userFetchedCallback)
		}

		if len(reviewerLogins) > 0 {
			var writer *csv.Writer
			if repo2Config.csv != "" {
				path := fmt.Sprintf("%s_%s-%s_pr_reviewers.csv", repo2Config.csv, args[0], args[1])
				writer = MustInitCsv(path, true)
			}
			GetUsersByLogins(reviewerLogins, ctx, ghClient, writer, userFetchedCallback)
		}
	}
}

func userFetchedCallback(fetched UserFetchResult, ctx context.Context, client *githubv4.Client, csvWriter *csv.Writer) {
	if fetched.Err != nil {
		log.WithError(fetched.Err).Warn()
		return
	}
	if isLocationInteresting(string(fetched.User.Location)) {
		fmt.Printf("%q\n", fetched.User.FormatForCsv())
		if csvWriter != nil {
			csvWriter.Write(fetched.User.FormatForCsv())
		}
	} else if repo2Config.verbose {
		fmt.Fprintf(os.Stderr, "%s's \"%s\" location was not interesting\n",
			fetched.Login, fetched.User.Location)
	}
}

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

// GetUsersByLogins is blocking
func GetUsersByLogins(logins []string, ctx context.Context, client *githubv4.Client, writer *csv.Writer,
	fetchCallback func(fetched UserFetchResult, ctx context.Context, client *githubv4.Client, writer *csv.Writer)) {
	out := getUsersByLogins(logins, ctx, client, 5*time.Second)

	for i := 0; i < len(logins); i++ {
		select {
		case fetchedUser := <-out:
			fetchCallback(fetchedUser, ctx, client, writer)
		case <-time.After(10 * time.Second):
			fmt.Println("timeout")
		}
	}

	if writer != nil {
		writer.Flush()
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
	err := client.Query(ctx, &q, map[string]interface{}{"login": githubv4.String(login), "maxOrgs": githubv4.Int(3)})
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
