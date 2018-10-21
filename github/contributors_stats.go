package github

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/go-github/github"
	"time"
)

type ContributorsFetchResult struct {
	Chunk    []*github.Contributor
	Response *github.Response
	Err      error
}

type ContributorsCallback func(page int, call ContributorsFetchResult)

func (f *fetcher) ParseContributors(ctx context.Context, perPage int, timeout time.Duration, callback ContributorsCallback) error {
	pageGetter := func(page int, out chan<- ContributorsFetchResult, totalPages int) {
		opt := &github.ListContributorsOptions{ListOptions: github.ListOptions{Page: page, PerPage: perPage}}
		chunk, response, err := f.GetClient().Repositories.ListContributors(ctx, f.GetOwner(), f.GetRepo(), opt)
		out <- ContributorsFetchResult{chunk, response, err}
		if totalPages != 0 && totalPages == len(out) {
			close(out)
		}
	}

	resultsChan := make(chan ContributorsFetchResult)

	go pageGetter(1, resultsChan, 0)

	var firstPageResults ContributorsFetchResult
	select {
	case firstPageResults = <-resultsChan:
		callback(1, firstPageResults)
	case <-time.After(timeout):
		err := errors.New("timeout while fetching first page")
		callback(1, ContributorsFetchResult{Err: err})
		return err
	}

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
