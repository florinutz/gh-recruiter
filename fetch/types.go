package fetch

import (
	"strconv"

	"github.com/shurcooL/githubv4"
)

type repository struct {
	ID              *githubv4.String
	URL             *githubv4.URI
	Description     *githubv4.String
	HomepageURL     *githubv4.URI
	NameWithOwner   *githubv4.String
	PrimaryLanguage LangFragment
	Forks           forks      `graphql:"forks(first: $forksPerBatch, orderBy: {field: STARGAZERS, direction: DESC})"`
	PullRequests    prs        `graphql:"pullRequests(first: $prsPerBatch, orderBy: {field: UPDATED_AT, direction: DESC})"`
	Releases        releases   `graphql:"releases(first: $releasesPerBatch, orderBy: {field: CREATED_AT, direction: DESC})"`
	Stargazers      stargazers `graphql:"stargazers(first: $stargazersPerBatch, orderBy: {field: STARRED_AT, direction: DESC})"`
}

// LangFragment is a graphql fragment for language
type LangFragment struct {
	ID   *githubv4.ID
	Name *githubv4.String
}

// User represents a gh user's interesting data
type User struct {
	ID        githubv4.ID
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
		// 	ID          githubv4.ID
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
func (u User) FormatForCsv() (result []string) {
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

type prReview struct {
	Author struct {
		Login githubv4.String
	}
	LastEditedAt githubv4.DateTime
	URL          githubv4.URI
}

type prComment struct {
	Author struct {
		Login githubv4.String
	}
	LastEditedAt githubv4.DateTime
	URL          githubv4.URI
}

type prCommit struct {
	Commit struct {
		Additions githubv4.Int
		Deletions githubv4.Int
		Author    struct {
			User User
		}
		AuthoredDate githubv4.DateTime
		Status       struct {
			State githubv4.StatusState
		}
		URL githubv4.URI
	}
}

type pr struct {
	Commits struct {
		PageInfo pageInfo
		Nodes    []prCommit
	} `graphql:"commits(first: $prCommitsPerBatch)"`
	Comments struct {
		PageInfo pageInfo
		Nodes    []prComment
	} `graphql:"comments(first: $prCommentsPerBatch)"`
	Reviews struct {
		PageInfo pageInfo
		Nodes    []prReview
	} `graphql:"reviews(first: $prReviewsPerBatch)"`
}

type prs struct {
	PageInfo pageInfo
	Nodes    []pr
}

type releases struct {
	TotalCount *githubv4.Int
	Nodes      []struct {
		Author User
	}
}

type stargazers struct {
	TotalCount *githubv4.Int
	Edges      []struct {
		StarredAt *githubv4.DateTime
		Node      User
	}
}

type forks struct {
	TotalCount *githubv4.Int
	Nodes      []struct {
		CreatedAt *githubv4.DateTime
		Owner     struct {
			ID    *githubv4.ID
			Login *githubv4.String
			URL   *githubv4.URI
		}
	}
}

type rateLimit struct {
	Cost      *githubv4.Int
	Limit     *githubv4.Int
	Remaining *githubv4.Int
	ResetAt   *githubv4.DateTime
}

// QueryRepo wraps the query with rateLimit info
type QueryRepo struct {
	Repository repository `graphql:"repository(owner:$repositoryOwner,name:$repositoryName)"`
	RateLimit  rateLimit
}

// UserFetchResult is used when returning users in a chan
type UserFetchResult struct {
	Login string
	User  User
	Err   error
}

type pageInfo struct {
	EndCursor   githubv4.String
	HasNextPage githubv4.Boolean
}

type forkNodes []struct {
	Owner struct {
		Login string
	}
}

// PrWithData represents the PR and its data
type PrWithData struct {
	URL      githubv4.URI
	Title    githubv4.String
	Comments struct {
		Nodes []prComment
	} `graphql:"comments(first: $prItemsPerBatch)"`
	Reviews struct {
		Nodes []prReview
	} `graphql:"comments(first: $prItemsPerBatch)"`
	Commits struct {
		Nodes []prCommit
	} `graphql:"commits(first: $prCommitsPerBatch)"`
}
