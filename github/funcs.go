package github

import (
	"context"
	"time"
)

func (f *fetcher) GetFuncs(ctx context.Context,
	cfr func(page int, call ContributorsFetchResult),
	csfr func(page int, call ContributorsStatsFetchResult),
	ffr func(page int, call ForksFetchResult),
	sfr func(page int, call StargazersFetchResult),
) []func() {
	return []func(){
		func() {
			f.ParseContributors(ctx, 10, 5*time.Second, cfr)
		},
		// func() {
		// 	err := f.ParseContributorsStats(ctx, 10, 5*time.Second, csfr)
		// 	if err != nil {
		// 		log.Fatal(err)
		// 	}
		// },
		// func() {
		// 	f.ParseForks(ctx, 10, 5*time.Second, ffr)
		// },
		// func() {
		// 	f.ParseStargazers(ctx, 10, 5*time.Second, sfr)
		// },
	}
}
