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

func NewFetcher(client *github.Client, owner string, repo string) *fetcher {
	return &fetcher{client: client, owner: owner, repo: repo}
}

func (f *fetcher) ParseContributors(
	ctx context.Context,
	callback func(contributorsChunk []*github.Contributor) error,
) error {
	page := 0

	for true {
		page++

		chunk, response, err := f.client.Repositories.ListContributors(ctx, f.owner, f.repo,
			&github.ListContributorsOptions{Anon: "false", ListOptions: github.ListOptions{Page: page, PerPage: 200}})

		if err != nil {
			if IsRateLimitError(err) {
				return errors.Wrap(err, "critical error while fetching contributors")
			}
			continue
		}

		if err = callback(chunk); err != nil {
			return errors.Wrap(err, "contributors chunk error")
		}

		if page >= response.LastPage {
			break
		}
	}

	return nil
}

func (f *fetcher) ParseForks(
	ctx context.Context,
	callback func(reposChunk []*github.Repository) error,
) error {
	page := 0

	for true {
		page++

		chunk, response, err := f.client.Repositories.ListForks(ctx, f.owner, f.repo,
			&github.RepositoryListForksOptions{ListOptions: github.ListOptions{Page: page, PerPage: 200}})

		if err != nil {
			if IsRateLimitError(err) {
				return errors.Wrap(err, "problem fetching forks")
			}
			continue
		}

		if err = callback(chunk); err != nil {
			return errors.Wrap(err, "forks chunk error")
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
) error {
	page := 0

	for true {
		page++

		chunk, response, err := f.client.Activity.ListStargazers(ctx, f.owner, f.repo,
			&github.ListOptions{Page: page, PerPage: 200})

		if err != nil {
			if IsRateLimitError(err) {
				return errors.Wrap(err, "problem fetching stargazers")
			}
			continue
		}

		if err = callback(chunk); err != nil {
			return errors.Wrap(err, "stargazers chunk error")
		}

		if page >= response.LastPage {
			break
		}
	}

	return nil
}

func IsRateLimitError(err error) bool {
	_, isRateLimitError := err.(*github.RateLimitError)
	_, isAbuseRateLimitError := err.(*github.AbuseRateLimitError)

	return isRateLimitError || isAbuseRateLimitError
}
