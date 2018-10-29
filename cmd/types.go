package cmd

import "github.com/shurcooL/githubv4"

type LangFragment struct {
	Id   *githubv4.ID
	Name *githubv4.String
}

type UserFragment struct {
	Id        *githubv4.ID
	Bio       *githubv4.String
	Company   *githubv4.String
	CreatedAt *githubv4.DateTime
	Email     *githubv4.String
	Followers struct {
		TotalCount *githubv4.Int
	}
	Following struct {
		TotalCount *githubv4.Int
	}
	IsBountyHunter *githubv4.Boolean
	IsCampusExpert *githubv4.Boolean
	IsViewer       *githubv4.Boolean
	IsEmployee     *githubv4.Boolean
	IsHireable     *githubv4.Boolean
	Location       *githubv4.String
}

type PRReview struct {
	Author struct {
		Login *githubv4.String
	}
	LastEditedAt *githubv4.DateTime
	Url          *githubv4.URI
	State        *githubv4.PullRequestReviewState
}

type PRComment struct {
	Author struct {
		Login *githubv4.String
	}
	LastEditedAt *githubv4.DateTime
	Url          *githubv4.URI
}

type PRCommit struct {
	Commit struct {
		Additions *githubv4.Int
		Deletions *githubv4.Int
		Author    struct {
			User UserFragment
		}
		AuthoredDate *githubv4.DateTime
		Status       struct {
			State *githubv4.StatusState
		}
	}
}

type PR struct {
	Commits struct {
		TotalCount *githubv4.Int
		Nodes      []PRCommit
	} `graphql:"commits(first: $prCommitsPerBatch)"`
	Comments struct {
		TotalCount *githubv4.Int
		Nodes      []PRComment
	} `graphql:"comments(first: $prCommentsPerBatch)"`
	Reviews struct {
		TotalCount *githubv4.Int
		Nodes      []PRReview
	} `graphql:"reviews(first: $prReviewsPerBatch)"`
}

type PRs struct {
	TotalCount *githubv4.Int
	Nodes      []PR
}

type Releases struct {
	TotalCount *githubv4.Int
	Nodes      []struct {
		Author UserFragment
	}
}

type Stargazers struct {
	TotalCount *githubv4.Int
	Edges      []struct {
		StarredAt *githubv4.DateTime
		Node      UserFragment
	}
}

type Forks struct {
	TotalCount *githubv4.Int
	Nodes      []struct {
		CreatedAt *githubv4.DateTime
		Owner     struct {
			Id    *githubv4.ID
			Login *githubv4.String
			Url   *githubv4.URI
		}
	}
}

type RateLimit struct {
	Cost      *githubv4.Int
	Limit     *githubv4.Int
	Remaining *githubv4.Int
	ResetAt   *githubv4.DateTime
}