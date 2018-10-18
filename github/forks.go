package github

import (
	"context"
	"fmt"
	"github.com/google/go-github/github"
	"time"
)

type ForksFetchResult struct {
	Chunk    []*github.Repository
	Response *github.Response
	Err      error
}

type ForksCallback func(page int, call ForksFetchResult)

func (f *fetcher) ParseForks(ctx context.Context, perPage int, timeout time.Duration, callback ForksCallback) error {
	pageGetter := func(page int, out chan<- ForksFetchResult, totalPages int) {
		opt := &github.RepositoryListForksOptions{ListOptions: github.ListOptions{Page: page, PerPage: perPage}}
		chunk, response, err := f.GetClient().Repositories.ListForks(ctx, f.GetOwner(), f.GetRepo(), opt)
		out <- ForksFetchResult{chunk, response, err}
		if totalPages != 0 && totalPages == len(out) {
			close(out)
		}
	}

	resultsChan := make(chan ForksFetchResult)

	go pageGetter(1, resultsChan, 0)

	firstPageResults := <-resultsChan
	callback(1, firstPageResults)

	totalPages := firstPageResults.Response.LastPage
	if totalPages > 1 {
		for page := 2; page <= totalPages; page++ {
			go pageGetter(page, resultsChan, totalPages)
		}
	}

	for page := 2; page <= totalPages; page++ {
		select {
		case call := <-resultsChan:
			callback(page, call)
		case <-time.After(timeout):
			fmt.Println("timeout")
		}
	}

	return nil
}
