package cmd

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/csv"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/pkg/errors"

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
		Use:   "repo",
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
	cache, err := getCache(CacheBucketName)
	if err != nil {
		log.Warnf("Running with no cache: %s\n", err)
	}
	ctx := context.Background()
	oauthClient := oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: rootConfig.token}))
	ghClient := githubv4.NewClient(oauthClient)

	g := GithubFetcher{ghClient, cache}

	if repoConfig.withForkers {
		logins, err := g.GetForkers(ctx, args[0], args[1], (*githubv4.String)(nil), 100)
		if err != nil {
			log.WithError(err).Fatal()
		}

		var writer *csv.Writer
		if repoConfig.csv != "" {
			path := fmt.Sprintf("%s_%s-%s_forkers.csv", repoConfig.csv, args[0], args[1])
			writer = MustInitCsv(path, true)
		}
		g.GetUsersByLogins(logins, ctx, writer, userFetchedCallback)
	}

	if repoConfig.withPRs {
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
			g.GetUsersByLogins(commenterLogins, ctx, writer, userFetchedCallback)
		}

		if len(reviewerLogins) > 0 {
			var writer *csv.Writer
			if repoConfig.csv != "" {
				path := fmt.Sprintf("%s_%s-%s_pr_reviewers.csv", repoConfig.csv, args[0], args[1])
				writer = MustInitCsv(path, true)
			}
			g.GetUsersByLogins(reviewerLogins, ctx, writer, userFetchedCallback)
		}
	}
}

func getCache(bucketName string) (cache *Cache, err error) {
	if cacheDir, err := os.UserCacheDir(); err != nil {
		return nil, err
	} else {
		c, err := boltdbcache.New(filepath.Join(cacheDir, bucketName))
		if err != nil {
			return nil, err
		}
		cache = &Cache{
			Validity: 168 * time.Hour,
			Cache:    c,
		}
	}

	return
}

func userFetchedCallback(fetched UserFetchResult, ctx context.Context, csvWriter *csv.Writer) {
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
	Cache  *Cache
}

func (g *GithubFetcher) GetUser(ctx context.Context, login string) (User, error) {
	var q struct {
		User      User `graphql:"user(login:$login)"`
		RateLimit RateLimit
	}
	vars := map[string]interface{}{"login": githubv4.String(login), "maxOrgs": githubv4.Int(3)}

	err := g.Query(ctx, &q, vars)
	if err != nil {
		return User{}, err
	}

	return q.User, nil
}

type Cache struct {
	httpcache.Cache
	Validity time.Duration
}

func (cache Cache) WriteQuery(q interface{}, variables map[string]interface{}) error {
	hash, err := getHashForCall(q, variables)
	if err != nil {
		return errors.Wrap(err, "coultn't compute ghv4 call hash")
	}
	cacheKey := fmt.Sprintf("query-%s", hash)
	toMarshal := QueryWithTime{Time: time.Now(), Query: q}

	buf := bytes.NewBuffer([]byte{})
	encoder := gob.NewEncoder(buf)
	encoder.Encode(toMarshal)

	cache.Set(cacheKey, buf.Bytes())

	return nil
}

// Query wraps the client's query in order to cache it
func (g *GithubFetcher) Query(ctx context.Context, q interface{}, variables map[string]interface{}) error {
	if t := reflect.TypeOf(q); t.Kind() != reflect.Ptr {
		return errors.New("incoming query is not a pointer")
	}

	if g.Cache != nil {
		if itemFromCache, err := g.Cache.ReadQuery(q, variables); err == nil {
			vq := reflect.ValueOf(q).Elem()
			vi := reflect.ValueOf(itemFromCache)
			vq.Set(vi)
			// q = itemFromCache

			// reflect.ValueOf(q).Elem().Set(reflect.ValueOf(itemFromCache))
			fmt.Printf("inside func: %+v\n\n", itemFromCache)
			fmt.Printf("typeof q: %s\n typeof variables: %s\n typeof itemFromCache: %s\n\n",
				reflect.TypeOf(q),
				reflect.TypeOf(variables),
				reflect.TypeOf(itemFromCache),
			)
			return nil
		}
	}

	err := g.Client.Query(ctx, q, variables)
	if err != nil {
		return err
	}

	if g.Cache != nil {
		g.Cache.WriteQuery(q, variables)
	}

	return nil
}

type QueryWithTime struct {
	Time  time.Time
	Query interface{}
}

func (cache Cache) ReadQuery(q interface{}, variables map[string]interface{}) (
	interface{}, error) {
	hash, err := getHashForCall(q, variables)
	if err != nil {
		return nil, err
	}
	cacheKey := fmt.Sprintf("query-%s", hash)

	var wt QueryWithTime
	item, ok := cache.Get(cacheKey)
	if !ok {
		return nil, fmt.Errorf("no cache for key %s", cacheKey)
	}

	buf := bytes.NewBuffer([]byte{})
	buf.Write(item)

	decoder := gob.NewDecoder(buf)
	err = decoder.Decode(q)
	if err != nil {
		return nil, errors.Wrap(err, "cache unmarshaling error")
	}

	if time.Since(wt.Time) > cache.Validity {
		return nil, fmt.Errorf("no cache for key %s", cacheKey)
	}

	return wt.Query, nil
}

func getJson(v interface{}, indent string, forZeroVal bool) (string, error) {
	if v == nil {
		return "", errors.New("nil input")
	}

	if forZeroVal {
		// make v its 0 value so we get consistent hashes even for incoming values
		ptr := reflect.New(reflect.TypeOf(v))
		v = ptr.Elem().Interface()
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

func getHashForCall(q interface{}, variables map[string]interface{}) (string, error) {
	jsonQuery, err := getJson(q, "", true)
	if err != nil {
		return "", err
	}
	jsonVars, err := getJson(variables, "", false)
	if err != nil {
		return "", err
	}

	sum := md5.Sum([]byte(jsonQuery + jsonVars))

	return hex.EncodeToString(sum[:]), nil
}

// GetUsersByLogins is blocking
func (g *GithubFetcher) GetUsersByLogins(logins []string, ctx context.Context, writer *csv.Writer,
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
	User  User
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

func (g *GithubFetcher) GetPRs(
	ctx context.Context,
	repoOwner string,
	repoName string,
	after *githubv4.String,
	depth int,
) (results []PRWithData, err error) {
	const PrsPerBatch = 100

	variables := map[string]interface{}{
		"repositoryOwner":   githubv4.String(repoOwner),
		"repositoryName":    githubv4.String(repoName),
		"prsPerBatch":       githubv4.Int(PrsPerBatch),
		"prItemsPerBatch":   githubv4.Int(100),
		"prCommitsPerBatch": githubv4.Int(5), // a safe value so that we don't request too much data
		"maxOrgs":           githubv4.Int(5),
		"after":             after,
	}

	var q struct {
		Repository struct {
			PullRequests struct {
				PageInfo PageInfo
				Nodes    []PRWithData
			} `graphql:"pullRequests(after: $after, first: $prsPerBatch, orderBy: {field: UPDATED_AT, direction: DESC})"`
		} `graphql:"repository(owner:$repositoryOwner,name:$repositoryName)"`
		RateLimit RateLimit
	}

	err = g.Query(ctx, &q, variables)
	if err != nil {
		return
	}

	hasNextPage := bool(q.Repository.PullRequests.PageInfo.HasNextPage)
	endCursor := &q.Repository.PullRequests.PageInfo.EndCursor

	results = append(results, q.Repository.PullRequests.Nodes...)

	depth++
	isNotDeepEnough := depth*PrsPerBatch <= 200

	if hasNextPage && isNotDeepEnough {
		data, err := g.GetPRs(ctx, repoOwner, repoName, endCursor, depth)
		if err != nil {
			return results, err
		}
		results = append(results, data...)
	}

	return
}

func (g *GithubFetcher) GetForkers(
	ctx context.Context,
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

	err = g.Query(ctx, &q, map[string]interface{}{
		"repositoryOwner": githubv4.String(repoOwner),
		"repositoryName":  githubv4.String(repoName),
		"itemsPerBatch":   githubv4.Int(pageSize),
		"after":           after,
	})
	fmt.Printf("outside func: %+v\n", q)
	if err != nil {
		return
	}

	for _, owner := range q.Repository.Forks.Nodes {
		results = append(results, owner.Owner.Login)
	}

	if !q.Repository.Forks.PageInfo.HasNextPage {
		return
	}

	after = &q.Repository.Forks.PageInfo.EndCursor

	data, err := g.GetForkers(ctx, repoOwner, repoName, after, pageSize)
	if err != nil {
		return results, err
	}

	results = append(results, data...)

	return
}
