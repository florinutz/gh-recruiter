package github

import (
	"context"
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

type ParserCallbacks struct {
	ChunkProcessor func(chunk []interface{}) error
	Fetcher        func(f *fetcher, ctx context.Context, page int) ([]interface{}, *github.Response, error)
}

func (f *fetcher) parse(
	ctx context.Context,
	callbacks ParserCallbacks,
) error {
	page := 0

	for true {
		page++

		chunk, response, err := callbacks.Fetcher(f, ctx, page)

		if err != nil {
			if IsRateLimitError(err) {
				return errors.Wrap(err, "problem fetching chunk")
			}
			continue
		}

		if err = callbacks.ChunkProcessor(chunk); err != nil {
			return errors.Wrap(err, "chunk processing error")
		}

		if page >= response.LastPage {
			break
		}
	}

	return nil
}

func (f *fetcher) ParseStargazers(
	ctx context.Context,
	callback func(stargazer []*github.Stargazer) error,
) (err error) {
	return f.parse(ctx, ParserCallbacks{
		Fetcher: func(f *fetcher, ctx context.Context, page int) (results []interface{}, response *github.Response, err error) {
			chunk, response, err := f.GetClient().Activity.ListStargazers(ctx, f.GetOwner(), f.GetRepo(),
				&github.ListOptions{Page: page, PerPage: 200})
			if err != nil {
				return nil, response, err
			}

			for _, element := range chunk {
				results = append(results, interface{}(element))
			}

			return
		},
		ChunkProcessor: func(chunk []interface{}) error {
			var elementsChunk []*github.Stargazer
			for _, el := range chunk {
				if element, ok := el.(*github.Stargazer); !ok {
					return errors.New("one in chunk wasn't a stargazer")
				} else {
					elementsChunk = append(elementsChunk, element)
				}
			}

			callback(elementsChunk)

			return nil
		},
	})
}

func (f *fetcher) ParseContributors(
	ctx context.Context,
	callback func(contributorsChunk []*github.Contributor) error,
) error {
	return f.parse(ctx, ParserCallbacks{
		Fetcher: func(f *fetcher, ctx context.Context, page int) (results []interface{}, response *github.Response, err error) {
			chunk, response, err := f.GetClient().Repositories.ListContributors(ctx, f.GetOwner(), f.GetRepo(),
				&github.ListContributorsOptions{Anon: "false", ListOptions: github.ListOptions{Page: page, PerPage: 200}})
			if err != nil {
				return nil, response, err
			}

			for _, element := range chunk {
				results = append(results, interface{}(element))
			}

			return
		},
		ChunkProcessor: func(chunk []interface{}) error {
			var elementsChunk []*github.Contributor
			for _, el := range chunk {
				if element, ok := el.(*github.Contributor); !ok {
					return errors.New("one in chunk wasn't a contributor")
				} else {
					elementsChunk = append(elementsChunk, element)
				}
			}

			callback(elementsChunk)

			return nil
		},
	})
}

func (f *fetcher) ParseForks(
	ctx context.Context,
	callback func(reposChunk []*github.Repository) error,
) error {
	return f.parse(ctx, ParserCallbacks{
		Fetcher: func(f *fetcher, ctx context.Context, page int) (results []interface{}, response *github.Response, err error) {
			chunk, response, err := f.GetClient().Repositories.ListForks(ctx, f.GetOwner(), f.GetRepo(),
				&github.RepositoryListForksOptions{ListOptions: github.ListOptions{Page: page, PerPage: 200}})
			if err != nil {
				return nil, response, err
			}

			for _, element := range chunk {
				results = append(results, interface{}(element))
			}

			return
		},
		ChunkProcessor: func(chunk []interface{}) error {
			var elementsChunk []*github.Repository
			for _, el := range chunk {
				if element, ok := el.(*github.Repository); !ok {
					return errors.New("one in chunk wasn't a fork")
				} else {
					elementsChunk = append(elementsChunk, element)
				}
			}

			callback(elementsChunk)

			return nil
		},
	})
}

func IsRateLimitError(err error) bool {
	_, isRateLimitError := err.(*github.RateLimitError)
	_, isAbuseRateLimitError := err.(*github.AbuseRateLimitError)

	return isRateLimitError || isAbuseRateLimitError
}
