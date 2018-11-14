package fetch

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"time"

	"github.com/florinutz/gh-recruiter/cache"
	"github.com/shurcooL/githubv4"
)

// GithubFetcher provides caching for a github graphql client's queries
type GithubFetcher struct {
	Client *githubv4.Client
	Cache  *cache.Cache
}

// GetUser retrieves a gh user
func (g *GithubFetcher) GetUser(ctx context.Context, login string) (User, error) {
	var q struct {
		User      User `graphql:"user(login:$login)"`
		RateLimit rateLimit
	}
	vars := map[string]interface{}{"login": githubv4.String(login), "maxOrgs": githubv4.Int(3)}

	err := g.Query(ctx, &q, vars)
	if err != nil {
		return User{}, err
	}

	return q.User, nil
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

// GetUsersByLogins retrieves users referenced by their logins
func (g *GithubFetcher) GetUsersByLogins(ctx context.Context, logins []string, writer *csv.Writer,
	fetchCallback func(ctx context.Context, fetched UserFetchResult, writer *csv.Writer)) {
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
			fetchCallback(ctx, fetchedUser, writer)
		case <-time.After(10 * time.Second):
			fmt.Println("timeout")
		}
	}

	if writer != nil {
		writer.Flush()
	}
}

// GetPRs returns PRs together with their interesting data
func (g *GithubFetcher) GetPRs(ctx context.Context, repoOwner string, repoName string, after *githubv4.String,
	depth int) (results []PrWithData, err error) {
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
				PageInfo pageInfo
				Nodes    []PrWithData
			} `graphql:"pullRequests(after: $after, first: $prsPerBatch, orderBy: {field: UPDATED_AT, direction: DESC})"`
		} `graphql:"repository(owner:$repositoryOwner,name:$repositoryName)"`
		RateLimit rateLimit
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

// GetForkers gets forkers for the repo
func (g *GithubFetcher) GetForkers(ctx context.Context, repoOwner string, repoName string, after *githubv4.String,
	pageSize int) (results []string, err error) {
	var q struct {
		Repository struct {
			Forks struct {
				PageInfo pageInfo
				Nodes    forkNodes
			} `graphql:"forks(first: $itemsPerBatch, after: $after, orderBy: {field: STARGAZERS, direction: DESC})"`
		} `graphql:"repository(owner:$repositoryOwner,name:$repositoryName)"`
		RateLimit rateLimit
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
