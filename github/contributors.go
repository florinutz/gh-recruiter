package github

import (
	"context"
	"github.com/google/go-github/github"
	"time"
)

type ContributorsStatsCallback func(page int, call ContributorsStatsFetchResult)

type ContributorsStatsFetchResult struct {
	Chunk    []*github.ContributorStats
	Response *github.Response
	Err      error
}

// no pagination here, simpler implementation
func (f *fetcher) ParseContributorsStats(ctx context.Context, perPage int, timeout time.Duration,
	callback ContributorsStatsCallback) error {
	chunk, response, err := f.GetClient().Repositories.ListContributorsStats(ctx, f.GetOwner(), f.GetRepo())
	if err != nil {
		return err
	}
	callback(1, ContributorsStatsFetchResult{chunk, response, err})

	return nil
}
