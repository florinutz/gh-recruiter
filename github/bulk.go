package github

import (
	"context"
	"fmt"
	"github.com/google/go-github/github"
	"sync"
	"time"
)

type fetcher struct {
	client *github.Client
	owner  string
	repo   string
}

func (f *fetcher) GetRepo() string {
	return f.repo
}

func (f *fetcher) GetOwner() string {
	return f.owner
}

func (f *fetcher) GetClient() *github.Client {
	return f.client
}

func NewFetcher(client *github.Client, owner string, repo string) *fetcher {
	return &fetcher{client: client, owner: owner, repo: repo}
}

type ForksFetchResult struct {
	chunk    []*github.Repository
	response *github.Response
	err      error
}

func (f *fetcher) ParseForks(ctx context.Context, perPage int, timeout time.Duration) error {
	pageGetter := func(page int, out chan<- ForksFetchResult, totalPages int, mutex *sync.Mutex) {
		opt := &github.RepositoryListForksOptions{ListOptions: github.ListOptions{Page: page, PerPage: perPage}}
		chunk, response, err := f.GetClient().Repositories.ListForks(ctx, f.GetOwner(), f.GetRepo(), opt)
		out <- ForksFetchResult{chunk, response, err}
		if totalPages != 0 && totalPages == len(out) {
			close(out)
		}
	}

	resultsChan := make(chan ForksFetchResult)

	mutex := &sync.Mutex{}

	go pageGetter(1, resultsChan, 0, mutex)

	firstPageResults := <-resultsChan
	parseForksFetchResult(1, firstPageResults)

	totalPages := firstPageResults.response.LastPage
	if totalPages > 1 {
		for page := 2; page <= totalPages; page++ {
			go pageGetter(page, resultsChan, totalPages, mutex)
		}
	}

	for page := 1; page < totalPages; page++ {
		select {
		case call := <-resultsChan:
			parseForksFetchResult(page, call)
		case <-time.After(timeout):
			fmt.Println("timeout")
		}
	}

	return nil
}

func parseForksFetchResult(page int, call ForksFetchResult) {
	err := call.err
	if err != nil {
		if IsRateLimitError(err) {
			fmt.Printf("rate limit hit while fetching page %d\n", page)
		} else {
			fmt.Printf("problem fetching page %d\n", page)
		}
	}
	for _, repo := range call.chunk {
		fmt.Printf("%s\n", *repo.URL)
	}
}

func IsRateLimitError(err error) bool {
	_, isRateLimitError := err.(*github.RateLimitError)
	_, isAbuseRateLimitError := err.(*github.AbuseRateLimitError)

	return isRateLimitError || isAbuseRateLimitError
}
