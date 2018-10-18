package github

import (
	"context"
	"fmt"
	"github.com/google/go-github/github"
	"time"
)

type StargazersFetchResult struct {
	Chunk    []*github.Stargazer
	Response *github.Response
	Err      error
}

type StargazersCallback func(page int, call StargazersFetchResult)

func (f *fetcher) ParseStargazers(ctx context.Context, perPage int, timeout time.Duration,
	callback StargazersCallback) error {
	pageGetter := func(page int, out chan<- StargazersFetchResult, totalPages int) {
		opt := &github.ListOptions{Page: page, PerPage: perPage}
		chunk, response, err := f.GetClient().Activity.ListStargazers(ctx, f.GetOwner(), f.GetRepo(), opt)
		out <- StargazersFetchResult{chunk, response, err}
		if totalPages != 0 && totalPages == len(out) {
			close(out)
		}
	}

	resultsChan := make(chan StargazersFetchResult)

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
