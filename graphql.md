```
{
  repository(owner: "hashicorp", name: "hcl") {
    id
    description
    forkCount
    homepageUrl
    nameWithOwner
    primaryLanguage {
      ...langFields
    }
    languages(first: 10, orderBy: {field: SIZE, direction: DESC}) {
      totalCount
      edges {
        node {
          ...langFields
        }
      }
    }
    forks(first: 1, orderBy: {field: STARGAZERS, direction: DESC}) {
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
    pullRequests(first: 3, orderBy: {field: UPDATED_AT, direction: DESC}) {
      totalCount
      edges {
        node {
          commits(first: 10) {
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
          comments(first: 10) {
            edges {
              node {
                author {
                  login
                }
              }
            }
          }
          reviews(first: 10) {
            edges {
              node {
                state
              }
            }
          }
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
  organizations(first: 15) {
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
  pinnedRepositories(first: 10) {
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

```