package github

import (
	"context"
	"fmt"
	"github.com/google/go-github/github"
	"github.com/pkg/errors"
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

func (f *fetcher) ParseForks(ctx context.Context) error {
	type fetchResult struct {
		chunk    []*github.Repository
		response *github.Response
		err      error
	}

	const chunkSize = 10

	results := make(chan fetchResult)
	lastPage := make(chan int)

	go func(out chan<- fetchResult, lastPage chan int) {
		for {
			go func(page int) {
				chunk, response, err := f.GetClient().Repositories.ListForks(ctx, f.GetOwner(), f.GetRepo(),
					&github.RepositoryListForksOptions{ListOptions: github.ListOptions{Page: page, PerPage: chunkSize}})
				out <- fetchResult{chunk, response, err}
			}(1)
		}
		close(out)

		// page := 0
		// for true {
		// 	page++
		// 	chunk, response, err := f.GetClient().Repositories.ListForks(ctx, f.GetOwner(), f.GetRepo(),
		// 		&github.RepositoryListForksOptions{ListOptions: github.ListOptions{Page: page, PerPage: 10}})
		//
		// 	fetchResult := fetchResult{chunk, response, err}
		//
		// 	out <- fetchResult
		//
		// 	if page >= response.LastPage {
		// 		close(out)
		// 		break
		// 	}
		// }
	}(results, lastPage)

	for fetchResult := range results {
		err := fetchResult.err
		if err != nil {
			if IsRateLimitError(err) {
				return errors.Wrap(err, "problem fetching chunk")
			}
		}
		for _, repo := range fetchResult.chunk {
			fmt.Printf("%s\n", *repo.URL)
		}
	}

	return nil
}

func IsRateLimitError(err error) bool {
	_, isRateLimitError := err.(*github.RateLimitError)
	_, isAbuseRateLimitError := err.(*github.AbuseRateLimitError)

	return isRateLimitError || isAbuseRateLimitError
}
