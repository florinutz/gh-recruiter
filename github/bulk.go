package github

import (
	"github.com/google/go-github/github"
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

func IsRateLimitError(err error) bool {
	_, isRateLimitError := err.(*github.RateLimitError)
	_, isAbuseRateLimitError := err.(*github.AbuseRateLimitError)

	return isRateLimitError || isAbuseRateLimitError
}
