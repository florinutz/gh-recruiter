package cmd

import (
	"strconv"

	"github.com/shurcooL/githubv4"
)

type Repository struct {
	Id              *githubv4.String
	Url             *githubv4.URI
	Description     *githubv4.String
	HomepageUrl     *githubv4.URI
	NameWithOwner   *githubv4.String
	PrimaryLanguage LangFragment
	Forks           Forks      `graphql:"forks(first: $forksPerBatch, orderBy: {field: STARGAZERS, direction: DESC})"`
	PullRequests    PRs        `graphql:"pullRequests(first: $prsPerBatch, orderBy: {field: UPDATED_AT, direction: DESC})"`
	Releases        Releases   `graphql:"releases(first: $releasesPerBatch, orderBy: {field: CREATED_AT, direction: DESC})"`
	Stargazers      Stargazers `graphql:"stargazers(first: $stargazersPerBatch, orderBy: {field: STARRED_AT, direction: DESC})"`
}

type LangFragment struct {
	Id   *githubv4.ID
	Name *githubv4.String
}

type UserFragment struct {
	Id        githubv4.ID
	Login     githubv4.String
	Location  githubv4.String
	Email     githubv4.String
	Name      githubv4.String
	Company   githubv4.String
	Bio       githubv4.String
	CreatedAt githubv4.DateTime
	Followers struct {
		TotalCount githubv4.Int
	}
	Following struct {
		TotalCount githubv4.Int
	}
	Organizations struct {
		TotalCount githubv4.Int
		// Nodes      []struct {
		// 	Id          githubv4.ID
		// 	Login       githubv4.String
		// 	Email       *githubv4.String
		// 	WebsiteUrl  *githubv4.URI
		// 	Description githubv4.String
		// }
	} `graphql:"organizations(first: $maxOrgs)"`
	IsBountyHunter githubv4.Boolean
	IsCampusExpert githubv4.Boolean
	IsViewer       githubv4.Boolean
	IsEmployee     githubv4.Boolean
	IsHireable     githubv4.Boolean
}

// FormatForCsv returns a []string representation for the full user
func (u UserFragment) FormatForCsv() (result []string) {
	result = []string{
		string(u.Login),
		string(u.Location),
		string(u.Email),
		string(u.Name),
		string(u.Company),
		string(u.Bio),
		string(u.CreatedAt.Format("02-Jan-2006")),
		strconv.Itoa(int(u.Followers.TotalCount)),
		strconv.Itoa(int(u.Following.TotalCount)),
		strconv.Itoa(int(u.Organizations.TotalCount)),
		strconv.FormatBool(bool(u.IsHireable)),
	}

	return
}

type PRReview struct {
	Author struct {
		Login githubv4.String
	}
	LastEditedAt githubv4.DateTime
	Url          githubv4.URI
}

type PRComment struct {
	Author struct {
		Login githubv4.String
	}
	LastEditedAt githubv4.DateTime
	Url          githubv4.URI
}

type PRCommit struct {
	Commit struct {
		Additions githubv4.Int
		Deletions githubv4.Int
		Author    struct {
			User UserFragment
		}
		AuthoredDate githubv4.DateTime
		Status       struct {
			State githubv4.StatusState
		}
		Url githubv4.URI
	}
}

type PR struct {
	Commits struct {
		PageInfo PageInfo
		Nodes    []PRCommit
	} `graphql:"commits(first: $prCommitsPerBatch)"`
	Comments struct {
		PageInfo PageInfo
		Nodes    []PRComment
	} `graphql:"comments(first: $prCommentsPerBatch)"`
	Reviews struct {
		PageInfo PageInfo
		Nodes    []PRReview
	} `graphql:"reviews(first: $prReviewsPerBatch)"`
}

type PRs struct {
	PageInfo PageInfo
	Nodes    []PR
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

type QueryRepo struct {
	Repository Repository `graphql:"repository(owner:$repositoryOwner,name:$repositoryName)"`
	RateLimit  RateLimit
}
