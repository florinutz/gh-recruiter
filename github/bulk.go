package github

import (
	"github.com/google/go-github/github"
	"github.com/gregjones/httpcache"
)

type fetcher struct {
	client *github.Client
	owner  string
	repo   string
	cache  httpcache.Cache
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

func NewFetcher(client *github.Client, owner string, repo string, cache httpcache.Cache) *fetcher {
	return &fetcher{client: client, owner: owner, repo: repo, cache: cache}
}

func IsRateLimitError(err error) bool {
	_, isRateLimitError := err.(*github.RateLimitError)
	_, isAbuseRateLimitError := err.(*github.AbuseRateLimitError)

	return isRateLimitError || isAbuseRateLimitError
}
