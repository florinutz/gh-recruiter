package cmd

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha1"
	"encoding/csv"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/birkelund/boltdbcache"
	"github.com/gregjones/httpcache"
	"github.com/shurcooL/githubv4"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

// repoCmd represents the repo command
var (
	repoCmd = &cobra.Command{
		Use:   "repo2",
		Short: "filters users who interacted with the repo by location",
		Run:   RunRepo,
		Args:  cobra.ExactArgs(2),
	}
)

var repoConfig struct {
	csv     string
	verbose bool

	withForkers bool
	withPRs     bool
}

func init() {
	repoCmd.Flags().StringVarP(&repoConfig.csv, "output", "o", "", "csv output file")
	repoCmd.Flags().BoolVarP(&repoConfig.verbose, "verbose", "v", false, "verbose?")
	repoCmd.Flags().BoolVarP(&repoConfig.withForkers, "forkers", "f", false, "fetch forkers?")
	repoCmd.Flags().BoolVarP(&repoConfig.withPRs, "prs", "p", false,
		"fetch users involved in PRs?")

	rootCmd.AddCommand(repoCmd)
}

const CacheBucketName = "gh-recruiter"

func RunRepo(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	cache, err := getCache(CacheBucketName)
	if err != nil {
		log.Warnf("Running with no cache: %s\n", err)
	}

	oauthClient := oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: rootConfig.token}))
	ghClient := githubv4.NewClient(oauthClient)

	if repoConfig.withForkers {
		logins, err := getForkers(ctx, ghClient, args[0], args[1], "", 100)
		if err != nil {
			log.WithError(err).Fatal()
		}

		var writer *csv.Writer
		if repoConfig.csv != "" {
			path := fmt.Sprintf("%s_%s-%s_forkers.csv", repoConfig.csv, args[0], args[1])
			writer = MustInitCsv(path, true)
		}
		GetUsersByLogins(logins, ctx, ghClient, writer, cache, userFetchedCallback)
	}

	if repoConfig.withPRs {
		prs, err := getPRs(ctx, ghClient, args[0], args[1], "", 0)
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
				if repoConfig.csv != "" {
					path := fmt.Sprintf("%s_%s-%s_pr_commits.csv", repoConfig.csv, args[0], args[1])
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
			if repoConfig.csv != "" {
				path := fmt.Sprintf("%s_%s-%s_pr_commenters.csv", repoConfig.csv, args[0], args[1])
				writer = MustInitCsv(path, true)
			}
			GetUsersByLogins(commenterLogins, ctx, ghClient, writer, cache, userFetchedCallback)
		}

		if len(reviewerLogins) > 0 {
			var writer *csv.Writer
			if repoConfig.csv != "" {
				path := fmt.Sprintf("%s_%s-%s_pr_reviewers.csv", repoConfig.csv, args[0], args[1])
				writer = MustInitCsv(path, true)
			}
			GetUsersByLogins(reviewerLogins, ctx, ghClient, writer, cache, userFetchedCallback)
		}
	}
}

func getCache(bucketName string) (cache httpcache.Cache, err error) {
	if cacheDir, err := os.UserCacheDir(); err != nil {
		return nil, err
	} else if cache, err = boltdbcache.New(filepath.Join(cacheDir, bucketName)); err != nil {
		return nil, err
	}

	return
}

func userFetchedCallback(ctx context.Context, fetched UserFetchResult, csvWriter *csv.Writer) {
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
	} else if repoConfig.verbose {
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

type GithubFetcher struct {
	Client *githubv4.Client
	Cache  *httpcache.Cache
}

func (g *GithubFetcher) GetUser(ctx context.Context, login string) (UserFragment, error) {
	var (
		buf      *bytes.Buffer
		cacheKey string
	)

	if g.Cache != nil {
		cache := *g.Cache
		h := md5.New()
		io.WriteString(h, login)
		cacheKey = fmt.Sprintf("user-%s", fmt.Sprintf("%x", h.Sum(nil)))

		if encoded, ok := cache.Get(cacheKey); ok {
			buf = bytes.NewBuffer(encoded)
			dec := gob.NewDecoder(buf)
			var u UserFragment
			err := dec.Decode(&u)
			if err != nil {
				log.WithError(err).Warn()
			}
			return u, err
		}
	}

	var q struct {
		User      UserFragment `graphql:"user(login:$login)"`
		RateLimit RateLimit
	}
	vars := map[string]interface{}{"login": githubv4.String(login), "maxOrgs": githubv4.Int(3)}

	err := g.Query(ctx, &q, vars)
	if err != nil {
		return UserFragment{}, err
	}

	if g.Cache != nil {
		cache := *g.Cache
		buf = bytes.NewBuffer(nil)
		enc := gob.NewEncoder(buf)
		err := enc.Encode(q.User)
		if err != nil {
			log.WithError(err).Warn()
		}
		cache.Set(cacheKey, buf.Bytes())
	}

	return q.User, nil
}

// Query wraps the client's query in order to cache it
func (g *GithubFetcher) Query(ctx context.Context, q interface{}, variables map[string]interface{}) error {
	var (
		cache    httpcache.Cache
		cacheKey string
	)

	type withTime struct {
		Time time.Time
		Q    interface{}
	}

	if g.Cache != nil {
		cache = *g.Cache
		hash, err := getHashForCall(q, variables)
		cacheKey = fmt.Sprintf("query-%s", string(hash))
		if err == nil {
			var withTime withTime
			if item, ok := cache.Get(cacheKey); ok {
				if err = json.Unmarshal(item, q); err == nil {
					return nil
				}
			}
		}
	}

	err := g.Client.Query(ctx, q, variables)
	if err != nil {
		return err
	}

	if g.Cache != nil {
		toMarshal := withTime{Time: time.Now(), Q: q}
		jsonBytes, err := json.Marshal(toMarshal)
		if err != nil {
			log.WithError(err).Warn()
		}
		cache.Set(cacheKey, jsonBytes)
	}

	return nil
}

func getJson(v interface{}, indent string) (string, error) {
	if v == nil {
		return "", errors.New("nil input")
	}

	var buf bytes.Buffer
	w := json.NewEncoder(&buf)
	w.SetIndent("", indent)
	err := w.Encode(v)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func getHashForCall(q interface{}, variables map[string]interface{}) ([]byte, error) {
	json1, err := getJson(q, "")
	if err != nil {
		return nil, err
	}

	json2, err := getJson(variables, "")
	if err != nil {
		return nil, err
	}

	return sha1.Sum([]byte(json1 + json2))[:], nil
}

// GetUsersByLogins is blocking
func (g *GithubFetcher) GetUsersByLogins(ctx context.Context, logins []string, writer *csv.Writer,
	fetchCallback func(fetched UserFetchResult, ctx context.Context, writer *csv.Writer)) {
	out := make(chan UserFetchResult)
	sent := 0
	for _, login := range logins {
		duration := time.Duration(rand.Intn(len(logins))) * time.Second // about 1 per second but randomly
		go func(login string, wait time.Duration) {
			time.Sleep(wait)

			user, err := g.GetUser(ctx, login)

			out <- UserFetchResult{login, user, err}
			sent++
			if sent == len(logins) {
				close(out)
			}
		}(login, duration)
	}

	for i := 0; i < len(logins); i++ {
		select {
		case fetchedUser := <-out:
			fetchCallback(fetchedUser, ctx, writer)
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

type UserFetchResult struct {
	Login string
	User  UserFragment
	Err   error
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
	depth int,
) (results []PRWithData, err error) {
	var (
		nodes       []PRWithData
		hasNextPage bool
		endCursor   string
	)

	const PrsPerBatch = 100

	variables := map[string]interface{}{
		"repositoryOwner":   githubv4.String(repoOwner),
		"repositoryName":    githubv4.String(repoName),
		"prsPerBatch":       githubv4.Int(PrsPerBatch),
		"prItemsPerBatch":   githubv4.Int(100),
		"prCommitsPerBatch": githubv4.Int(5), // a safe value so that we don't request too much data
		"maxOrgs":           githubv4.Int(5),
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

	depth++
	isNotDeepEnough := depth*PrsPerBatch <= 200

	if hasNextPage && isNotDeepEnough {
		data, err := getPRs(ctx, client, repoOwner, repoName, endCursor, depth)
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
	after *githubv4.String,
	pageSize int,
) (results []string, err error) {
	var q struct {
		Repository struct {
			Forks struct {
				PageInfo PageInfo
				Nodes    ForkNodes
			} `graphql:"forks(first: $itemsPerBatch, after: $after, orderBy: {field: STARGAZERS, direction: DESC})"`
		} `graphql:"repository(owner:$repositoryOwner,name:$repositoryName)"`
		RateLimit RateLimit
	}

	err = client.Query(ctx, &q, map[string]interface{}{
		"repositoryOwner": githubv4.String(repoOwner),
		"repositoryName":  githubv4.String(repoName),
		"itemsPerBatch":   githubv4.Int(pageSize),
		"after":           after,
	})
	if err != nil || !q.Repository.Forks.PageInfo.HasNextPage {
		return
	}

	for _, owner := range q.Repository.Forks.Nodes {
		results = append(results, owner.Owner.Login)
	}

	after = &q.Repository.Forks.PageInfo.EndCursor

	data, err := getForkers(ctx, client, repoOwner, repoName, after, pageSize)
	if err != nil {
		return results, err
	}

	results = append(results, data...)

	return
}
