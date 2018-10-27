package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"os"

	"github.com/shurcooL/githubv4"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

// repoCmd represents the repo command
var repo2Cmd = &cobra.Command{
	Use:   "repo2",
	Short: "filters users who interacted with the repo by location",
	Run:   RunRepo2,
	Args:  cobra.ExactArgs(2),
}

var repo2Config struct {
	location string
	csv      bool
}

func init() {
	repo2Cmd.Flags().StringVarP(&repo2Config.location, "location", "l", "", "location filter")
	repo2Cmd.Flags().BoolVarP(&repo2Config.csv, "csv", "c", false, "csv output")

	rootCmd.AddCommand(repo2Cmd)
}

type LangFragment struct {
	Id   githubv4.ID
	Name githubv4.String
}

type UserFragment struct {
	Id        githubv4.ID
	Bio       githubv4.String
	Company   githubv4.String
	CreatedAt githubv4.DateTime
	Email     githubv4.String
	Followers struct {
		TotalCount githubv4.Int
	}
	Following struct {
		TotalCount githubv4.Int
	}
	IsBountyHunter githubv4.Boolean
	IsCampusExpert githubv4.Boolean
	IsViewer       githubv4.Boolean
	IsEmployee     githubv4.Boolean
	IsHireable     githubv4.Boolean
	Location       githubv4.String
}

type PRs struct {
	TotalCount githubv4.Int
	Nodes      []struct {
		Commits struct {
			TotalCount githubv4.Int
			Nodes      []struct {
				Commit struct {
					Additions githubv4.Int
					Deletions githubv4.Int
					Author    struct {
						User UserFragment
					}
					AuthoredDate githubv4.Date
					Status       struct {
						State githubv4.StatusState
					}
				}
			}
		} `graphql:"commits(first: $prCommitsPerBatch)"`
		Comments struct {
			TotalCount githubv4.Int
			Nodes      []struct {
				Author struct {
					Login githubv4.String
				}
			}
		} `graphql:"comments(first: $prCommentsPerBatch)"`
	}
}

func RunRepo2(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	oauthClient := oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: rootConfig.token}))
	githubGraphQlClient := githubv4.NewClient(oauthClient)

	var crawlRepoQuery struct {
		Repository struct {
			Id              githubv4.String
			Url             githubv4.URI
			Description     githubv4.String
			HomepageUrl     githubv4.URI
			NameWithOwner   githubv4.String
			PrimaryLanguage LangFragment
			Forks           struct {
				TotalCount githubv4.Int
				Nodes      []struct {
					CreatedAt githubv4.Date
					Owner     struct {
						Id    githubv4.ID
						Login githubv4.String
						Url   githubv4.URI
					}
				}
			} `graphql:"forks(first: $forksPerBatch, orderBy: {field: STARGAZERS, direction: DESC})"`
			PullRequests PRs `graphql:"pullRequests(first: $prsPerBatch, orderBy: {field: UPDATED_AT, direction: DESC})"`
		} `graphql:"repository(owner:$repositoryOwner,name:$repositoryName)"`
		RateLimit struct {
			Cost      githubv4.Int
			Limit     githubv4.Int
			Remaining githubv4.Int
			ResetAt   githubv4.DateTime
		}
	}

	variables := map[string]interface{}{
		"repositoryOwner":    githubv4.String(args[0]),
		"repositoryName":     githubv4.String(args[1]),
		"forksPerBatch":      githubv4.Int(3),
		"prsPerBatch":        githubv4.Int(3),
		"prCommitsPerBatch":  githubv4.Int(3),
		"prCommentsPerBatch": githubv4.Int(3),
	}

	err := githubGraphQlClient.Query(ctx, &crawlRepoQuery, variables)
	if err != nil {
		log.Fatalln(err)
	}
	printJSON(crawlRepoQuery)
}

// printJSON prints v as JSON encoded with indent to stdout. It panics on any error.
func printJSON(v interface{}) {
	if v == nil {
		log.WithError(errors.New("nil value for json")).Fatal()
	}
	w := json.NewEncoder(os.Stdout)
	w.SetIndent("", "\t")
	err := w.Encode(v)
	if err != nil {
		log.WithError(err).Fatal()
	}
}
