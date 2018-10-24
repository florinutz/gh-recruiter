query crawlRepo($owner: String!, $repo: String!, $first: Int!, $maxStargazers: Int = 10, $maxForks: Int = 10, $maxPRs: Int = 10, $maxReleases: Int = 10, $userMaxOrgs: Int = 10, $userMaxPinnedRepos: Int = 10) {
  repository(owner: $owner, name: $repo) {
    id
    description
    forkCount
    homepageUrl
    nameWithOwner
    primaryLanguage {
      ...langFields
    }
    forks(first: $maxForks, orderBy: {field: STARGAZERS, direction: DESC}) {
      totalCount
      edges {
        node {
          createdAt
          databaseId
          forkCount
          homepageUrl
          id
          isLocked
          isMirror
          isPrivate
        }
      }
    }
    pullRequests(first: $maxPRs, orderBy: {field: UPDATED_AT, direction: DESC}) {
      totalCount
      edges {
        node {
          commits(first: $first) {
            edges {
              node {
                commit {
                  additions
                  deletions
                  author {
                    user {
                      ...userFields
                    }
                  }
                  authoredByCommitter
                  authoredDate
                  status {
                    state
                  }
                }
              }
            }
          }
          comments(first: $first) {
            edges {
              node {
                author {
                  login
                }
              }
            }
          }
          reviews(first: $first) {
            edges {
              node {
                state
              }
            }
          }
        }
      }
    }
    releases(first: $maxReleases, orderBy: {field: CREATED_AT, direction: DESC}) {
      totalCount
      edges {
        node {
          author {
            ...userFields
          }
        }
      }
    }
    stargazers(first: $maxStargazers, orderBy: {field: STARRED_AT, direction: DESC}) {
      totalCount
      edges {
        starredAt
        node {
          ...userFields
        }
      }
    }
  }
}

fragment langFields on Language {
  id
  name
}

fragment userFields on User {
  id
  bio
  company
  createdAt
  email
  followers {
    totalCount
  }
  following {
    totalCount
  }
  isBountyHunter
  isCampusExpert
  isViewer
  isEmployee
  isHireable
  location
  issueComments {
    totalCount
  }
  login
  name
  organizations(first: $userMaxOrgs) {
    edges {
      node {
        id
        websiteUrl
        description
        email
        location
        login
      }
    }
  }
  pinnedRepositories(first: $userMaxPinnedRepos) {
    edges {
      node {
        name
        description
        forkCount
        homepageUrl
        issues {
          totalCount
        }
        primaryLanguage {
          ...langFields
        }
        pullRequests {
          totalCount
        }
        releases {
          totalCount
        }
        stargazers {
          totalCount
        }
        updatedAt
        url
        watchers {
          totalCount
        }
      }
    }
  }
}


{
  "owner": "hashicorp",
  "repo": "hcl",
  "first": 2
}